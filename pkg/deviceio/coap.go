package deviceio

//
// Copyright 2020 Telenor Digital AS
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"github.com/ExploratoryEngineering/logging"
	"github.com/eesrc/horde/pkg/deviceio/rxtx"
	"github.com/go-ocf/go-coap"
	"github.com/go-ocf/go-coap/codes"
)

const (
	// coapDeviceTimeout is the timeout for responses to the CoAP servers running
	// on devices. The latency in the network can be quite bad.
	coapTimeout = 90 * time.Minute

	// coapDefaultPort is the default port. This is the plain UDP port. DTLS
	// uses a different port.
	coapDefaultPort = 5683

	// Max payload size for CoAP, in bytes. This is just a guesstimate. Some
	// firmware images might be bigger but we'll get an error when this happens.
	coapMaxPayloadSize = 1024 * 1024

	// This is the error class code for CoAP (b01000000). Since all errors have
	// the 2nd bit set all values above this is errors
	coapErrorCode = codes.Code(0x80)
)

// CoAPServer is the CoAP listener and server. It proxies requests to the gRPC
// upstream service. The upstream service handles all logic.
type CoAPServer struct {
	server      *coap.Server
	config      CoAPParameters
	client      rxtx.RxtxClient
	terminate   *int32
	backlogger  messageBacklog
	clientConns *ttlMap
	inError     *int32
	naslist     []int32
}

// NewCoAPServer creates a new CoAP listener service. The service exposes
// a (single) CoAP endpoint to devices.
func NewCoAPServer(client rxtx.RxtxClient, config CoAPParameters) *CoAPServer {
	ret := &CoAPServer{
		config:      config,
		client:      client,
		terminate:   nil,
		clientConns: newTTLMap(defaultClientTimeout),
		inError:     new(int32),
		naslist:     make([]int32, 0),
	}
	// Prepopulate the NAS list to use in requests
	for _, v := range config.NASList() {
		ret.naslist = append(ret.naslist, int32(v))
	}
	atomic.StoreInt32(ret.inError, 0)
	return ret
}

// Start the server. It can only be started once.
func (c *CoAPServer) Start() error {
	if c.terminate != nil {
		return errors.New("already started")
	}
	c.terminate = new(int32)
	atomic.StoreInt32(c.terminate, 0)
	mux := coap.NewServeMux()
	mux.DefaultHandleFunc(c.defaultHandler)
	c.server = &coap.Server{
		Addr:    c.config.Endpoint,
		Net:     c.config.Protocol,
		Handler: mux,
	}
	var err error
	c.backlogger, err = newMessageBacklog(backlogCoAPDatabase)
	if err != nil {
		return err
	}
	errCh := make(chan error)
	go func(errCh chan error) {
		logging.Info("CoAP server listens on %s. APN ID=%d NAS ID=%v Protocol=%s",
			c.config.Endpoint, c.config.APNID, c.config.NASList(), c.config.Protocol)
		errCh <- c.server.ListenAndServe()
	}(errCh)

	go c.pushPoller()

	// Push backlog messages when we start
	pushed := 0
	logging.Info("Pushing backlog messages...")
	for c.pushBacklog() {
		pushed++
		time.Sleep(1 * time.Microsecond)
	}
	logging.Info("%d messages from backlog has been pushed", pushed)

	select {
	case err := <-errCh:
		return err
	case <-time.After(250 * time.Millisecond):
		return nil
	}
}

// Helper function to set the origin field in the requests
func (c *CoAPServer) origin() *rxtx.Origin {
	return &rxtx.Origin{
		ApnId: int32(c.config.APNID),
		NasId: c.naslist,
	}
}

// Stop shuts down the CoAP transceiver service.
func (c *CoAPServer) Stop() {
	if err := c.server.Shutdown(); err != nil {
		logging.Warning("Error shutting down CoAP server: %v", err)
	}
	atomic.StoreInt32(c.terminate, 1)
}

// Convert token in a byte buffer into an signed 64-bit integer. The token can
// be anything from 1 to 8 bytes
func (c *CoAPServer) getToken(token []byte) int64 {
	switch len(token) {
	case 8:
		return int64(binary.LittleEndian.Uint64(token))
	case 4:
		return int64(binary.LittleEndian.Uint32(token))
	case 2:
		return int64(binary.LittleEndian.Uint16(token))
	case 1:
		return int64(token[0])
	default:
		return 0
	}
}

// isRequest returns true if the message is a request initiated by the other
// side. For some weird reason this isn't implemented by the coap library
func (c *CoAPServer) isRequest(code codes.Code) bool {
	if code == codes.GET || code == codes.DELETE || code == codes.POST || code == codes.PUT {
		return true
	}
	return false
}

// The default handler for CoAP requests (from the devices)
func (c *CoAPServer) defaultHandler(w coap.ResponseWriter, r *coap.Request) {
	if c.config.AuditLog {
		logging.Info("Request from %s: Method=%s Path=%s, Payload=%d bytes, Location=%+v",
			r.Client.RemoteAddr().String(), r.Msg.Code().String(), r.Msg.PathString(), len(r.Msg.Payload()), r.Msg.Option(coap.LocationPath))
	}

	// Ship off to upstream service
	udpAddr, ok := (r.Client.RemoteAddr()).(*net.UDPAddr)
	if !ok {
		logging.Warning("Request with non-UDP address to CoAP server: %s. Ignoring request.", r.Client.RemoteAddr().String())
		return
	}

	// Cache the connection for later. CoAP clients wants to receive data on the
	// samme connection. We
	c.clientConns.AddClientConnection(udpAddr.String(), r.Client)

	// Construct the upstream message
	msg := rxtx.Message{
		Type:          rxtx.MessageType_CoAPUpstream,
		RemoteAddress: udpAddr.IP,
		RemotePort:    int32(udpAddr.Port),
		Payload:       r.Msg.Payload(),
		Coap: &rxtx.CoAPOptions{
			Code:     int32(r.Msg.Code()),
			Type:     int32(r.Msg.Type()),
			Path:     r.Msg.PathString(),
			UriQuery: r.Msg.Query(),
			Token:    c.getToken(r.Msg.Token()),
		},
	}
	upstream := &upstreamData{Msg: msg}

	// Add to backlog in case we can't send it.
	if err := c.backlogger.Add(upstream); err != nil {
		logging.Warning("Got error adding message to backlog: %v", err)
	}

	// We'll only send responses when the client expects it. Confirmations can
	// be piggybacked on new requests and the clients *must*  (cf spec) handle
	// this but I can't say for certain that the library authors have read that
	// part of the spec so it will be handled separately.

	req := &rxtx.UpstreamRequest{
		Origin:           c.origin(),
		Redelivery:       false,
		ExpectDownstream: c.isRequest(r.Msg.Code()),
		Msg:              &msg,
	}

	// Send message to the upstream server.
	res, err := c.sendUpstreamWithRetry(req)
	if err != nil {
		logging.Debug("Could not send upstream. Stashing in backlog")
		c.backlogger.CancelRemove(upstream)
		logging.Error("Unable to send CoAP request to upstream service. Sending blank, valid response (err=%s)", err)
		c.sendBlankResponse(w, r)
		return
	}

	// Message is shipped to server, remove from backlog
	c.backlogger.ConfirmRemove(upstream)

	hasResponse := res != nil && res.Msg != nil && res.Msg.Coap != nil
	if !hasResponse {
		// If there is no response to the client send a blank message if the
		// message is confirmable.
		if r.Msg.IsConfirmable() {
			logging.Debug("Confirm with blank response to %s", r.Client.RemoteAddr().String())
			c.sendBlankResponse(w, r)
			return
		}
		logging.Debug("Non confirmable response to %s and no message upstream (%s). Returning.", r.Msg.Path(), r.Client.RemoteAddr().String())
		return
	}

	// Invariant: We have a response and we should send it.
	co := codes.Code(res.Msg.Coap.Code)
	logging.Debug("Sending response with ID=%d code=%s", res.Msg.Id, co.String())

	downMsg := w.NewResponse(co)

	downMsg.SetMessageID(r.Msg.MessageID())
	downMsg.SetToken(r.Msg.Token())

	if res.Msg.Coap.Type != 0 {
		downMsg.SetType(coap.COAPType(res.Msg.Coap.Type))
	}

	if res.Msg.Coap.ContentFormat != 0 {
		downMsg.SetOption(coap.ContentFormat, coap.MediaType(res.Msg.Coap.ContentFormat))
	}

	if len(res.Msg.Coap.LocationPath) > 0 {
		downMsg.SetOption(coap.LocationPath, res.Msg.Coap.LocationPath)
	}

	downMsg.SetPayload(res.Msg.Payload)

	// Set timeout for message. Use the default if it isn't set.
	timeout := coapTimeout
	if res.Msg.Coap.TimeoutSeconds != 0 {
		timeout = time.Duration(res.Msg.Coap.TimeoutSeconds) * time.Second
	}

	ctx, done := context.WithTimeout(context.Background(), timeout)
	defer done()

	if c.config.AuditLog {
		logging.Info("Sending %d byte to %s/%s", len(downMsg.Payload()), r.Client.RemoteAddr().String(), downMsg.PathString())
	}

	if err := w.WriteMsgWithContext(ctx, downMsg); err != nil {
		logging.Warning("Error writing response dest=%s, path=%s payload=%d: %v", udpAddr.String(), downMsg.PathString(), len(downMsg.Payload()), err)
		sendAckWithRetry(c.client, res.Msg.Id, rxtx.ErrorCode_NOT_HANDLED)
		return
	}
	// Update the client map since it might have expired at this time.
	c.clientConns.AddClientConnection(udpAddr.String(), r.Client)
	// Send ack on payload if it is set
	sendAckWithRetry(c.client, res.Msg.Id, rxtx.ErrorCode_SUCCESS)
}

// Poll for push messages from the service. Push messages are sent without a
// corresponding request from the client.
func (c *CoAPServer) pushPoller() {
	for {
		if atomic.LoadInt32(c.terminate) == 1 {
			return
		}
		res, err := c.getPush()
		if err != nil {
			time.Sleep(sleepOnError)
			continue
		}
		if res.Msg == nil {
			c.pushBacklog()
			time.Sleep(sleepOnEmpty)
			continue
		}
		logging.Debug("Sending push message: %s coap://%s:%d%s, payload=%d bytes", codes.Code(res.Msg.Coap.Code).String(), net.IP(res.Msg.RemoteAddress).String(), res.Msg.RemotePort, res.Msg.Coap.Path, len(res.Msg.Payload))
		go c.sendPushMessage(*res.Msg)
	}
}

// Send push messages to the devices.
func (c *CoAPServer) sendPushMessage(msg rxtx.Message) {
	if msg.RemotePort == 0 {
		// use the default CoAP port
		logging.Debug("Remote port not set. Using default CoAP port (%d) for message", coapDefaultPort)
		msg.RemotePort = coapDefaultPort
	}

	if msg.Coap == nil || msg.RemoteAddress == nil {
		logging.Warning("Incomplete message. Can't send (%+v)", msg)
		c.sendAck(msg.Id, rxtx.ErrorCode_PARAMETER)
		return
	}

	ip := net.IP(msg.RemoteAddress)
	endpoint := fmt.Sprintf("%s:%d", ip.String(), msg.RemotePort)

	if len(msg.Payload) > coapMaxPayloadSize {
		logging.Warning("Payload is too large (%d bytes). Rejecting message to %s", len(msg.Payload), endpoint)
		c.sendAck(msg.Id, rxtx.ErrorCode_TOO_LARGE)
		return
	}

	conn := c.clientConns.GetConnection(endpoint)
	if conn == nil {
		logging.Debug("Creating NEW connection to %s", endpoint)
		var err error
		conn, err = coap.Dial("udp", endpoint)
		if err != nil {
			logging.Warning("Could not dial to %s. Returning error: %v", endpoint, err)
			c.sendAck(msg.Id, rxtx.ErrorCode_NETWORK)
			return
		}
		defer conn.Close()
	}
	logging.Debug("Local address = %s, remote address = %s", conn.LocalAddr().String(), conn.RemoteAddr().String())
	params := coap.MessageParams{
		Code:      codes.Code(msg.Coap.Code),
		Type:      coap.COAPType(msg.Coap.Type),
		MessageID: coap.GenerateMessageID(),
	}

	if msg.Coap.Token != 0 {
		token := make([]byte, 8)
		binary.LittleEndian.PutUint64(token, uint64(msg.Coap.Token))
		params.Token = token[:]
		logging.Debug("Using existing token %d", msg.Coap.Token)
	} else {
		logging.Debug("Token is 0, generating a new token")
		var err error
		params.Token, err = coap.GenerateToken()
		if err != nil {
			logging.Warning("Could not generate a token for push message: %v", err)
		}
	}

	params.Payload = msg.Payload
	cm := coap.NewDgramMessage(params)
	timeout := coapTimeout
	if msg.Coap.TimeoutSeconds != 0 {
		timeout = time.Duration(msg.Coap.TimeoutSeconds) * time.Second
	}

	cm.SetOption(coap.ContentFormat, coap.MediaType(msg.Coap.ContentFormat))
	if msg.Coap.Accept != 0 {
		cm.SetOption(coap.Accept, coap.MediaType(msg.Coap.Accept))
	}
	if len(msg.Coap.UriQuery) > 0 {
		cm.SetOption(coap.URIQuery, msg.Coap.UriQuery)
	}
	cm.SetPathString(msg.Coap.Path)
	ctx, done := context.WithTimeout(context.Background(), timeout)
	defer done()

	res, err := conn.ExchangeWithContext(ctx, cm)
	if err != nil {
		logging.Warning("Send error for CoAP message to coap://%s/%s: %v", endpoint, msg.Coap.Path, err)
		if err == context.DeadlineExceeded {
			c.sendAck(msg.Id, rxtx.ErrorCode_TIMEOUT)
			return
		}
		c.sendAck(msg.Id, rxtx.ErrorCode_NETWORK)
		return
	}
	if res.Code() >= coapErrorCode {
		c.sendAck(msg.Id, rxtx.ErrorCode_CLIENT_ERROR)
	} else {
		c.sendAck(msg.Id, rxtx.ErrorCode_SUCCESS)
	}

	_, err = c.sendUpstreamWithRetry(&rxtx.UpstreamRequest{
		Origin:           c.origin(),
		ExpectDownstream: false,
		Msg: &rxtx.Message{
			RemoteAddress: msg.RemoteAddress,
			RemotePort:    msg.RemotePort,
			Type:          rxtx.MessageType_CoAPUpstream,
			Payload:       res.Payload(),
			Coap: &rxtx.CoAPOptions{
				Token: c.getToken(res.Token()),
				Path:  res.PathString(),
				Code:  int32(res.Code()),
			},
		},
	})
	if err != nil {
		logging.Warning("Could not send response to push message: %v", err)
	}
}

func (c *CoAPServer) sendAck(msgID int64, result rxtx.ErrorCode) {
	ctx, done := context.WithTimeout(context.Background(), grpcTimeout)
	defer done()

	req := &rxtx.AckRequest{
		MessageId: msgID,
		Result:    result,
	}
	_, err := c.client.Ack(ctx, req)
	if err != nil {
		// Do a retry
		ctx2, done2 := context.WithTimeout(context.Background(), grpcRetryTimeout)
		defer done2()
		_, err = c.client.Ack(ctx2, req)
		if err != nil {
			logging.Warning("Error sending ack (msg id=%d, result=%s) to upstream server: %v", msgID, result.String(), err)
		}
	}
}

func (c *CoAPServer) getPush() (*rxtx.DownstreamResponse, error) {
	ctx, done := context.WithTimeout(context.Background(), grpcTimeout)
	defer done()

	req := &rxtx.DownstreamRequest{
		Origin: c.origin(),
		Type:   rxtx.MessageType_CoAPPush,
	}
	res, err := c.client.GetMessage(ctx, req)
	if err != nil {
		if atomic.LoadInt32(c.inError) == 0 {
			logging.Warning("Got error retrieving push messages: %v", err)
		}
		atomic.StoreInt32(c.inError, 1)
		return nil, err
	}
	if atomic.LoadInt32(c.inError) == 1 {
		logging.Info("Rxtx service is back up again")
		atomic.StoreInt32(c.inError, 0)
	}
	return res, err
}

func (c *CoAPServer) sendUpstreamWithRetry(req *rxtx.UpstreamRequest) (*rxtx.DownstreamResponse, error) {
	ctx, done := context.WithTimeout(context.Background(), grpcTimeout)
	defer done()
	res, err := c.client.PutMessage(ctx, req)
	if err != nil {
		ctx2, done2 := context.WithTimeout(context.Background(), grpcRetryTimeout)
		defer done2()
		res, err = c.client.PutMessage(ctx2, req)
	}
	return res, err
}

func (c *CoAPServer) pushBacklog() bool {
	if m := c.backlogger.Get(false); m != nil {
		req := &rxtx.UpstreamRequest{
			ExpectDownstream: false,
			Origin:           c.origin(),
			Redelivery:       true,
			Msg:              &m.Msg,
		}
		ctx, done := context.WithTimeout(context.Background(), grpcTimeout)
		defer done()
		// The response is discarded here but we won't process any replies
		// and the expect downstream flag is set to false.
		_, err := c.client.PutMessage(ctx, req)
		if err != nil {
			c.backlogger.CancelRemove(m)
			return false
		}
		c.backlogger.ConfirmRemove(m)
		return true
	}
	return false
}

func (c *CoAPServer) sendBlankResponse(w coap.ResponseWriter, r *coap.Request) {
	if !r.Msg.IsConfirmable() {
		return
	}
	responseCode := codes.Valid
	switch r.Msg.Code() {
	case codes.POST:
		// Zephyr CoAP library likes a 2.04 Created response when POSTing
		responseCode = codes.Created
	case codes.PUT:
		responseCode = codes.Created
	case codes.GET:
		responseCode = codes.Valid
	case codes.DELETE:
		responseCode = codes.Deleted
	default:
		responseCode = codes.Valid
	}

	msg := w.NewResponse(responseCode)
	msg.SetMessageID(r.Msg.MessageID())
	msg.SetToken(r.Msg.Token())
	ctx, done := context.WithTimeout(context.Background(), coapTimeout)
	defer done()
	if err := w.WriteMsgWithContext(ctx, msg); err != nil {
		logging.Warning("Error writing blank response to %s: %v", r.Client.RemoteAddr().String(), err)
	}
}
