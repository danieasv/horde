package radius

//
//Copyright 2019 Telenor Digital AS
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.
//
import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/ExploratoryEngineering/logging"
	"github.com/eesrc/horde/pkg/utils/audit"
	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
)

// Server is the RADIUS server
type Server interface {
	// Start launches the server. The supplied ranges is
	// the ranges the server will handle
	Start() error
	// Stop stops the server
	Stop() error
	// Address is the local address of the server
	Address() string
}

type radiusServer struct {
	server *radius.PacketServer
	config ServerParameters
	done   chan bool
	serve  AccessRequestHandlerFunc
	addr   net.Addr
}

// NewRADIUSServer creates a new RADIUS server
func NewRADIUSServer(config ServerParameters, handler AccessRequestHandlerFunc) Server {
	return &radiusServer{
		config: config,
		serve:  handler,
		done:   make(chan bool),
	}
}

// Start the radius server. The supplied NAS identifiers are used for monitoring.
func (s *radiusServer) Start() error {
	if s.config.SharedSecret == "" || len(s.config.SharedSecret) == 0 {
		return errors.New("RADIUS shared secret must be set")
	}
	s.server = &radius.PacketServer{
		SecretSource: radius.StaticSecretSource([]byte(s.config.SharedSecret)),
		Handler:      radius.HandlerFunc(s.radiusHandler),
	}
	result := make(chan error)

	listener, err := net.ListenPacket("udp4", s.config.Endpoint)
	if err != nil {
		return err
	}
	s.addr = listener.LocalAddr()
	logging.Info("RADIUS server is listening on %s", s.addr.String())
	go func(result chan error) {
		err := s.server.Serve(listener)
		if err != nil && err != radius.ErrServerShutdown {
			logging.Warning("RADIUS server got error listening: %v", err)
			result <- err
		}
		s.done <- true
	}(result)

	select {
	case err := <-result:
		return err
	case <-time.After(100 * time.Millisecond):
		break
	}

	return nil
}

// Stop the radius server.
func (s *radiusServer) Stop() error {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancelFunc()

	logging.Info("Shutting down RADIUS server")
	if err := s.server.Shutdown(ctx); err != nil {
		return err
	}

	select {
	case <-ctx.Done():
	case <-s.done:
	}

	return nil
}

// radiusHandler is the main handler for incoming radius packets.
// From this function we dispatch the request to the right internal
// handler.
func (s *radiusServer) radiusHandler(w radius.ResponseWriter, r *radius.Request) {
	switch r.Code {
	case radius.CodeAccessRequest:
		s.handleAccessRequest(w, r)

	case radius.CodeAccountingRequest:
		s.handleAccountingRequest(w, r)

	case radius.CodeDisconnectRequest:
		s.handleDisconnectRequest(w, r)

	default:
		logging.Warning("Unknown Radius request request=%+v", r)
	}
}

// handleAccessRequest takes care of parsing the Radius Access-Request
// and handing it to the supplied AccessRequestHandler.
func (s *radiusServer) handleAccessRequest(w radius.ResponseWriter, r *radius.Request) {
	// While the handler might not be specified, we still would like
	// our logs to contain a trace of who tried to log in, so we parse
	// the packet now so we can include the fields in log messages.
	accessRequest := accessRequestFromPacket(r.Packet)

	// AccessRequestHandler has not been set so we cannot accept any Access-Request
	if s.serve == nil {
		logging.Error("Rejecting request for device with IMSI=%s, NAS=%s (no handler registered)",
			accessRequest.IMSI,
			accessRequest.NASIdentifier)

		reply := r.Response(radius.CodeAccessReject)
		rfc2865.ReplyMessage_AddString(reply, HandlerNotRegisteredErrorMsg)
		w.Write(reply)
		return
	}

	response := s.serve(accessRequest)
	if !response.Accept {
		audit.Log("RADIUS: Rejecting device with IMSI=%s, NAS=%s", accessRequest.IMSI, accessRequest.NASIdentifier)
		reply := r.Response(radius.CodeAccessReject)
		rfc2865.ReplyMessage_AddString(reply, response.RejectMessage)
		w.Write(reply)
		return
	}

	// Make sure the IP address returned by the AccessRequestHandler isn't nonsense
	if response.IPAddress.IsUnspecified() {
		response.RejectMessage = IPAddressInvalid

		logging.Error("IP address from request handlers is invalid. Rejecting device with IMSI %s (NAS=%s)", accessRequest.IMSI, accessRequest.NASIdentifier)

		reply := r.Response(radius.CodeAccessReject)
		rfc2865.ReplyMessage_AddString(reply, response.RejectMessage)
		w.Write(reply)
		return
	}

	audit.Log("RADIUS Accepting device with IMSI=%s NAS=%s, IP=%s", accessRequest.IMSI, accessRequest.NASIdentifier, response.IPAddress.String())
	// If we are here it means that a handler was registered, that the
	// AccessHandler decided to grant access and that we have a valid
	// IP address.
	//
	// TODO(borud): We should figure out if we want to send more
	// fields here.  For instance it may be nice to send a Framed-MTU
	// field to give the device a hint as to safe MTU.
	reply := r.Response(radius.CodeAccessAccept)
	rfc2865.FramedIPAddress_Add(reply, response.IPAddress)
	w.Write(reply)
}

func (s *radiusServer) handleAccountingRequest(w radius.ResponseWriter, r *radius.Request) {
	logging.Warning("Got Accounting-Request from %v, not implemented yet", r.RemoteAddr)
}

func (s *radiusServer) handleDisconnectRequest(w radius.ResponseWriter, r *radius.Request) {
	logging.Warning("Got Disconnect-Request from %v, not implemented yet", r.RemoteAddr)
}

func (s *radiusServer) Address() string {
	return s.addr.String()
}
