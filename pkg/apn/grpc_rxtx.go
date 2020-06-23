package apn

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
	"crypto/rand"
	"encoding/binary"
	"errors"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/ExploratoryEngineering/logging"
	"github.com/eesrc/horde/pkg/deviceio/rxtx"
	"github.com/eesrc/horde/pkg/metrics"
	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
	"github.com/go-ocf/go-coap/codes"
	"github.com/golang/protobuf/proto"
)

// ResponseCallback is used to send feedback to the CoAP handlers. It will be
// invoked when a response is returned.
type ResponseCallback func(rxtx.ErrorCode)

// CoAPHandler is a message handler for the CoAP service. The various CoAP services implement
// this handler to process requests (and responses) from devices. Replayed messages
// won't be processed by the handlers.
type CoAPHandler func(apnID int, nasID int,
	device *model.Device,
	request *rxtx.UpstreamRequest) (*rxtx.DownstreamResponse, ResponseCallback, error)

// Token listeners. This listens for CoAP tokens in upstream messages
type listenerCallback func(*rxtx.Message)

// Add a listener for upstream messages matching the token. Messages that are
// handled by callbacks are discarded, ie not included in the upstream messages
func (r *RxTxReceiver) addUpstreamListener(token int64, cb listenerCallback) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.tokenListeners[token] = cb
}

// remove listener for token.
func (r *RxTxReceiver) removeUpstreamListener(token int64) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.tokenListeners, token)
}

// RxTxReceiver is the gRPC server for the UDP and CoAP listeners.
type RxTxReceiver struct {
	apnStore        storage.APNStore
	store           storage.DataStore
	downstreamStore storage.DownstreamStore
	apnConfig       *storage.APNConfigCache
	coapHandlers    map[string]CoAPHandler
	publisher       chan<- model.DataMessage
	mutex           *sync.Mutex
	callbacks       map[int64]ResponseCallback
	tokenListeners  map[int64]listenerCallback
}

// NewRxTxReceiver is the API for the gRPC service that handles messages to and from the devices.
func NewRxTxReceiver(
	apnConfig *storage.APNConfigCache,
	store storage.DataStore,
	apnStore storage.APNStore,
	downstreamStore storage.DownstreamStore,
	publisher chan<- model.DataMessage) *RxTxReceiver {
	metrics.DefaultAPNCounters.Start(apnConfig)
	return &RxTxReceiver{
		apnConfig:       apnConfig,
		store:           store,
		apnStore:        apnStore,
		downstreamStore: downstreamStore,
		publisher:       publisher,
		coapHandlers:    make(map[string]CoAPHandler),
		mutex:           &sync.Mutex{},
		callbacks:       make(map[int64]ResponseCallback),
		tokenListeners:  make(map[int64]listenerCallback),
	}
}

// AddCoAPHandler adds a new handler. The handler is matched based on the
// *start* of the path, ie /foo matches /foo/bar, /foo, /foo/ but not /foobar
// set by the client. There is no wildcards in the path.
func (r *RxTxReceiver) AddCoAPHandler(path string, handler CoAPHandler) {
	if _, ok := r.coapHandlers[path]; ok {
		logging.Warning("Overwriting CoAP handler with path %s", path)
	}
	r.coapHandlers[path] = handler
}

// PutMessage is called by the listeners when they have received a message
func (r *RxTxReceiver) PutMessage(ctx context.Context, req *rxtx.UpstreamRequest) (*rxtx.DownstreamResponse, error) {
	if req.Msg == nil {
		metrics.DefaultAPNCounters.MessageError()
		logging.Warning("Msg field is not set. Discarding message")
		return &rxtx.DownstreamResponse{}, errors.New("needs message field")
	}

	if req.Msg.RemoteAddress == nil {
		metrics.DefaultAPNCounters.MessageError()
		logging.Warning("Remote address not set for message from %+v", req.Origin)
		return &rxtx.DownstreamResponse{}, errors.New("remote address not set")
	}

	ip := net.IP(req.Msg.RemoteAddress)
	if req.Origin == nil {
		metrics.DefaultAPNCounters.MessageError()
		logging.Warning("Origin field is not set. Discarding message (remote address=%s, type=%s)", ip.String(), req.Msg.Type.String())
		return &rxtx.DownstreamResponse{}, errors.New("needs origin field")
	}

	// Find out who handled it
	nasranges, ok := r.apnConfig.FindAPN(int(req.Origin.ApnId))
	if !ok {
		logging.Warning("Got message from unknown APN (%d). Discarding message", req.Origin.ApnId)
		metrics.DefaultAPNCounters.Rejected()
		return &rxtx.DownstreamResponse{}, errors.New("unknown APN ID")
	}

	metrics.DefaultAPNCounters.MessageReceived(nasranges)

	imsi, err := r.apnStore.LookupIMSIFromIP(ip, nasranges)
	if err != nil {
		metrics.DefaultAPNCounters.MessageError()
		metrics.DefaultAPNCounters.MessageRejected(nasranges)
		if err == storage.ErrNotFound {
			logging.Info("Got message from unknown device. Remote address=%s. Ignoring it.", ip.String())
			return &rxtx.DownstreamResponse{}, nil
		}
		logging.Warning("Error doing device lookup: %v", err)
		// If we return an error the listener will keep the message in the backlog.
		return &rxtx.DownstreamResponse{}, err
	}

	device, err := r.store.RetrieveDeviceByIMSI(imsi)
	if err != nil {
		metrics.DefaultAPNCounters.MessageError()
		metrics.DefaultAPNCounters.MessageRejected(nasranges)
		// TODO(stalehd): This might be too easy. Since there is an IP->IMSI mapping there might be
		// an issue with the service and the message should be stashed away.
		logging.Warning("Unable to retrieve device via IMSI (%d): %v. Ignoring message", imsi, err)
		return &rxtx.DownstreamResponse{}, nil
	}
	consistent := true
	if device.Network.AllocatedIP != ip.String() {
		consistent = false
	}
	if device.Network.ApnID != int(req.Origin.ApnId) {
		consistent = false
	}
	nas, ok := nasranges.ByIP(ip)
	if !ok {
		logging.Warning("Got message from unknown IP range (APN ID=%d, NAS ID=%d Remote address=%s). Setting NAS ID to existing value", nasranges.APN.ID, req.Origin.NasId, ip.String())
		nas.ID = device.Network.NasID
	}
	if nas.ID != device.Network.NasID {
		consistent = false
	}
	if !consistent {
		// This shouldn't really happen but when testing with external servers or
		// if something is broken in the RADIUS server the IP address might not
		// match. This *could* cause data leaks to devices when addresses are
		// recycled so we'll log an error here.
		logging.Error("Got request from IP=%s, APN=%d, NAS=%d but device (IMSI=%d) has IP=%s, APN=%d, NAS=%d. Updating device to reflect new reality",
			ip.String(), req.Origin.ApnId, nas.ID, device.IMSI,
			device.Network.AllocatedIP, device.Network.ApnID, device.Network.NasID)
		device.Network.AllocatedIP = ip.String()
		device.Network.AllocatedAt = time.Now()
		device.Network.ApnID = int(req.Origin.ApnId)
		device.Network.NasID = nas.ID
		if err := r.store.UpdateDeviceMetadata(device); err != nil {
			logging.Warning("Unable to update device's IP address (IMSI=%d IP=%s): %v", device.IMSI, device.Network.AllocatedIP, err)
		}
	}

	switch req.Msg.Type {
	case rxtx.MessageType_UDP:
		metrics.DefaultAPNCounters.In(nasranges.APN, nas, model.UDPTransport)
		return r.udpHandler(nasranges.APN.ID, nas.ID, &device, req.Redelivery, req.ExpectDownstream, req.Msg)

	case rxtx.MessageType_CoAPPush:
		metrics.DefaultAPNCounters.In(nasranges.APN, nas, model.CoAPTransport)
		fallthrough
	case rxtx.MessageType_CoAPPull:
		metrics.DefaultAPNCounters.In(nasranges.APN, nas, model.CoAPPullTransport)
		fallthrough
	case rxtx.MessageType_CoAPUpstream:
		resp, handler, err := r.coapHandler(nasranges.APN.ID, nas.ID, &device, req)
		if handler != nil {
			r.addCallbackListener(resp.Msg.Id, handler)
		}
		return resp, err
	default:
		metrics.DefaultAPNCounters.MessageRejected(nasranges)

		// Technically this shouldn't happen but...
		logging.Error("Got unknown message type: %s from %s:%d. Don't know how to process it",
			req.Msg.Type.String(), ip.String(), req.Msg.RemotePort)
		return &rxtx.DownstreamResponse{}, err
	}
}

func (r *RxTxReceiver) addCallbackListener(msgID int64, handler ResponseCallback) {
	r.mutex.Lock()
	r.callbacks[msgID] = handler
	r.mutex.Unlock()
}

// GetMessage is called by the listeners when they poll for new downstream messages.
func (r *RxTxReceiver) GetMessage(ctx context.Context, req *rxtx.DownstreamRequest) (*rxtx.DownstreamResponse, error) {
	// If this is the CoAP server the coap-pull requests are part of the putmessage request so this is just for coap-push transport
	transport := model.UDPTransport
	if req.Type == rxtx.MessageType_CoAPPush {
		transport = model.CoAPTransport
	}
	for _, nasid := range req.Origin.NasId {
		_, buf, err := r.downstreamStore.Retrieve(int(req.Origin.ApnId), int(nasid), transport)
		if err != nil {
			if err != storage.ErrNotFound {
				logging.Warning("Got error retrieving message from downstream store for APN ID %d NAS ID %d: %v.Returning empty response", req.Origin.ApnId, req.Origin.NasId, err)
				return &rxtx.DownstreamResponse{}, nil
			}
			continue
		}
		msg := &rxtx.Message{}
		if err := proto.Unmarshal(buf, msg); err != nil {
			logging.Warning("Unable to unmarshal protobuf message from downstream store: %v", err)
			return nil, err
		}
		nasrange, ok := r.apnConfig.FindAPN(int(req.Origin.ApnId))
		if ok {
			nas, ok := nasrange.Find(int(nasid))
			if ok {
				metrics.DefaultAPNCounters.Out(nasrange.APN, nas, transport)
			}
			metrics.DefaultAPNCounters.MessageForwarded(nasrange)
		}
		return &rxtx.DownstreamResponse{
			Msg: msg,
		}, nil
	}
	return &rxtx.DownstreamResponse{}, nil
}

// Ack is sent by the listeners when they have finished processing a downstream message
func (r *RxTxReceiver) Ack(ctx context.Context, req *rxtx.AckRequest) (*rxtx.AckResponse, error) {
	// This is a response to a  message. Check if one of the listeners are waiting for it
	r.mutex.Lock()
	handler, ok := r.callbacks[req.MessageId]
	if ok {
		delete(r.callbacks, req.MessageId)
	}
	r.mutex.Unlock()
	if ok {
		handler(req.Result)
	}

	// Remove the message from the store regardless of result. We won't do any
	// retries at this time.
	if err := r.downstreamStore.Delete(model.MessageKey(req.MessageId)); err != nil {
		if err != storage.ErrNotFound {
			logging.Warning("Got error removing message from downstream store (id=%d): %v", req.MessageId, err)
		}
	}
	return &rxtx.AckResponse{}, nil
}

// udpHandler handles incoming UDP messages. These are regular upstream messages and are passed on as is
func (r *RxTxReceiver) udpHandler(apnID int, nasID int, device *model.Device, redelivery bool, wantResponse bool, msg *rxtx.Message) (*rxtx.DownstreamResponse, error) {
	ts := time.Now()
	if redelivery {
		// use the time stamp from the listener
		ts = time.Unix(0, msg.Timestamp)
	}

	// All messages from UDP are published since they all originate at the device.
	r.publish(device, ts, msg)

	if redelivery {
		// No response if it is a redelivery
		return &rxtx.DownstreamResponse{}, nil
	}
	if !wantResponse {
		return &rxtx.DownstreamResponse{}, nil
	}

	_, buf, err := r.downstreamStore.RetrieveByDevice(device.ID, model.UDPPullTransport)
	if err != nil {
		if err != storage.ErrNotFound {
			logging.Warning("Error retrieving downstream message for IMSI %d: %v. Sending blank response.", device.IMSI, err)
		}
		return &rxtx.DownstreamResponse{}, nil
	}
	outMsg := &rxtx.Message{}
	if err := proto.Unmarshal(buf, outMsg); err != nil {
		logging.Warning("Error unmarshaling message from downstream store: %v", err)
		return nil, err
	}
	// if port is set to 0 we'll use any port. This is slightly breaking wrt the old API behaviour but
	// desirable.
	if outMsg.RemotePort == 0 || outMsg.RemotePort == msg.RemotePort {
		// Jolly good. Send it in return.
		return &rxtx.DownstreamResponse{
			Msg: outMsg,
		}, nil
	}

	// No outgoing message so just return empty response
	return &rxtx.DownstreamResponse{}, nil
}

// coapHandler dispatches the request depending on the path. If there is no
// matching handler it is handled as an upstream message
func (r *RxTxReceiver) coapHandler(apnID int, nasID int, device *model.Device, req *rxtx.UpstreamRequest) (*rxtx.DownstreamResponse, ResponseCallback, error) {
	if req.Msg.Coap == nil {
		logging.Warning("Got CoAP message from IMSI %d, APN=%d, NAS=%d but no CoAP options were set. Listener might be broken.", device.IMSI, apnID, nasID)
		return nil, nil, errors.New("missing CoAP options")
	}
	for path, handler := range r.coapHandlers {
		// Skip the leading slash from the path
		if path[0] == '/' {
			path = path[1:]
		}
		matchPath := req.Msg.Coap.Path

		if strings.HasPrefix(matchPath, path) {
			// If this is a redelivery the device might be gone. Drop the message
			if req.Redelivery {
				logging.Warning("Got redelivery of CoAP message to %s (from IMSI=%d). Dropping response", req.Msg.Coap.Path, device.IMSI)
				return &rxtx.DownstreamResponse{}, nil, nil
			}
			return handler(apnID, nasID, device, req)
		}
	}
	// This is the default handler - treat as upstream data message. The other handlers
	// will process (and discard) the other messages.
	ts := time.Now()
	if req.Redelivery {
		// use the time stamp from the listener
		ts = time.Unix(0, req.Msg.Timestamp)
	}
	upstream := true
	r.mutex.Lock()
	for token, handler := range r.tokenListeners {
		if token == req.Msg.Coap.Token {
			handler(req.Msg)
			upstream = false
		}
	}
	r.mutex.Unlock()

	if upstream && len(req.Msg.Payload) > 0 {
		r.publish(device, ts, req.Msg)
	}

	if req.Redelivery {
		// Redeliveries might not have a device waiting so just return a blank
		return &rxtx.DownstreamResponse{}, nil, nil
	}
	// If this isn't a GET message the client isn't expecting a payload in
	// return.
	if req.Msg.Coap.Code != int32(codes.GET) {
		// TODO: Check if there are other methods that returns a payload.
		return &rxtx.DownstreamResponse{}, nil, nil
	}

	if !req.ExpectDownstream {
		return &rxtx.DownstreamResponse{}, nil, nil
	}

	_, buf, err := r.downstreamStore.RetrieveByDevice(device.ID, model.CoAPPullTransport)
	if err != nil {
		if err != storage.ErrNotFound {
			logging.Warning("Error retrieving downstream message for IMSI %d: %v. Sending blank response.", device.IMSI, err)
		}
		return &rxtx.DownstreamResponse{}, nil, nil
	}
	outMsg := &rxtx.Message{}
	if err := proto.Unmarshal(buf, outMsg); err != nil {
		logging.Warning("Error unmarshaling downstream store message: %v", err)
		return nil, nil, err
	}
	if outMsg.Coap != nil {
		outMsg.Coap.Code = int32(codes.Content)
	}
	return &rxtx.DownstreamResponse{
		Msg: outMsg,
	}, nil, nil
}

// publish publishes the message internally in Horde
func (r *RxTxReceiver) publish(device *model.Device, ts time.Time, msg *rxtx.Message) {
	dm := model.DataMessage{
		Device:   *device,
		Received: ts,
		Payload:  msg.Payload,
	}
	switch msg.Type {
	case rxtx.MessageType_UDP:
		dm.Transport = model.UDPTransport
		dm.UDP = model.UDPMetaData{
			LocalPort:  int(msg.LocalPort),
			RemotePort: int(msg.RemotePort),
		}

	case rxtx.MessageType_CoAPUpstream:
		code := codes.Code(msg.Coap.Code)
		if code != codes.POST && code != codes.PUT {
			logging.Error("can't publish CoAP %s messages (%+v)", code.String(), msg)
			return
		}
		dm.Transport = model.CoAPPullTransport
		dm.CoAP = model.CoAPMetaData{
			Code: code.String(),
			Path: msg.Coap.Path,
		}
	default:
		logging.Error("can't publish %s messages", msg.Type.String())
		return
	}
	r.publisher <- dm
}

// Send is used to send a message to a device. It can be sent asynchronously.
func (r *RxTxReceiver) Send(ctx context.Context, device model.Device, msg *rxtx.Message, wait bool) (rxtx.ErrorCode, error) {
	if msg == nil {
		return rxtx.ErrorCode_CLIENT_ERROR, errors.New("no message to send")
	}

	if (msg.Type == rxtx.MessageType_CoAPPush || msg.Type == rxtx.MessageType_CoAPPull) && msg.Coap == nil {
		return rxtx.ErrorCode_CLIENT_ERROR, errors.New("needs coap options")
	}
	// Push messages needs port and address
	pushMessage := (msg.Type == rxtx.MessageType_CoAPPush || msg.Type == rxtx.MessageType_UDP)

	if pushMessage {
		if device.Network.AllocatedIP == "" {
			return rxtx.ErrorCode_CLIENT_ERROR, errors.New("device is not online")
		}
		if msg.RemotePort == 0 {
			return rxtx.ErrorCode_CLIENT_ERROR, errors.New("need port")
		}
	}
	if (msg.Type == rxtx.MessageType_UDP || msg.Type == rxtx.MessageType_CoAPPull) && len(msg.Payload) == 0 {
		return rxtx.ErrorCode_CLIENT_ERROR, errors.New("no payload")
	}

	var transport = model.UDPTransport
	switch msg.Type {
	case rxtx.MessageType_CoAPPull:
		transport = model.CoAPPullTransport
	case rxtx.MessageType_CoAPPush:
		transport = model.CoAPTransport
	case rxtx.MessageType_UDP:
		transport = model.UDPTransport
	case rxtx.MessageType_UDPPull:
		transport = model.UDPPullTransport
	default:
		return rxtx.ErrorCode_CLIENT_ERROR, errors.New("can't send that message type")
	}

	// Ship the message
	msgID := r.downstreamStore.NewMessageID()
	msg.Id = int64(msgID)

	buf, err := proto.Marshal(msg)
	if err != nil {
		return rxtx.ErrorCode_INTERNAL, err
	}
	if err := r.downstreamStore.Create(device.Network.ApnID, device.Network.NasID, device.ID,
		msgID, transport, buf); err != nil {
		return rxtx.ErrorCode_INTERNAL, err
	}

	if msg.Type == rxtx.MessageType_CoAPPull {
		return rxtx.ErrorCode_PENDING, nil
	}
	if msg.Type == rxtx.MessageType_UDPPull {
		return rxtx.ErrorCode_PENDING, nil
	}
	if wait {
		// wait for message
		resultCh := make(chan rxtx.ErrorCode)
		r.addCallbackListener(msg.Id, func(result rxtx.ErrorCode) {
			select {
			case <-ctx.Done():
				return
			default:
				resultCh <- result
			}
		})
		defer close(resultCh)
		select {
		case ret := <-resultCh:
			switch ret {
			case rxtx.ErrorCode_SUCCESS:
				r.notifySendSuccess(device.Network.ApnID)
			default:
				r.notifySendError(device.Network.ApnID)
			}
			return ret, nil
		case <-ctx.Done():
			r.notifySendError(device.Network.ApnID)
			return rxtx.ErrorCode_TIMEOUT, errors.New("timed out")
		}
	}
	return rxtx.ErrorCode_PENDING, nil
}

func (r *RxTxReceiver) notifySendError(apnID int) {
	nasrange, ok := r.apnConfig.FindAPN(apnID)
	if !ok {
		return
	}
	metrics.DefaultAPNCounters.MessageSendError(nasrange)
}

func (r *RxTxReceiver) notifySendSuccess(apnID int) {
	nasrange, ok := r.apnConfig.FindAPN(apnID)
	if !ok {
		return
	}
	metrics.DefaultAPNCounters.MessageSent(nasrange)
}

func (r *RxTxReceiver) createToken() (int64, error) {
	buf := make([]byte, 8)
	_, err := rand.Read(buf)
	if err != nil {
		return 0, err
	}
	return int64(binary.BigEndian.Uint64(buf)), nil
}

// Exchange does a CoAP exhange of messages. The exchange assumes that the
// response is returned to the same service as the one emitting the request, ie
// if there's a reshard midway through the exchange it will be null and void.
// TODO(stalehd): Context on request for timeouts.
func (r *RxTxReceiver) Exchange(ctx context.Context, device *model.Device, msg *rxtx.Message) (*rxtx.Message, error) {
	if msg.Type != rxtx.MessageType_CoAPPush {
		return nil, errors.New("must use coap push messages")
	}
	if msg == nil || msg.Coap == nil {
		return nil, errors.New("not a coap message")
	}

	msgID := r.downstreamStore.NewMessageID()
	msg.Id = int64(msgID)

	// Generate a downstream message and set the token. The token is unique
	// for this instance.
	token, err := r.createToken()
	if err != nil {
		logging.Error("Could not create token for exchange: %v", err)
		return nil, err
	}
	msg.Coap.Token = token

	msgChan := make(chan *rxtx.Message)
	defer close(msgChan)

	defer r.removeUpstreamListener(token)
	// Wait for the response. Listen for the token rather than the message ID
	r.addUpstreamListener(token, func(res *rxtx.Message) {
		// This might panic if the channel is closed at the same time as
		// a message is sent.
		defer recover()

		select {
		case msgChan <- res:
			// empty
		case <-ctx.Done():
			// empty
		}
	})

	buf, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}

	// Ship the message
	if err := r.downstreamStore.Create(device.Network.ApnID, device.Network.NasID, device.ID, msgID, model.CoAPTransport, buf); err != nil {
		logging.Error("Could not create downstream message for exchange: %v", err)
		return nil, err
	}
	select {
	case m := <-msgChan:
		r.downstreamStore.Delete(msgID)
		return m, nil
	case <-ctx.Done():
		// Remove the message since it timed out
		r.downstreamStore.Delete(msgID)
		return nil, errors.New("message timed out")
	}
}
