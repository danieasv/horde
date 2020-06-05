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
	"fmt"
	"math/rand"
	"net"
	"testing"
	"time"

	"github.com/ExploratoryEngineering/logging"
	"github.com/eesrc/horde/pkg/deviceio/rxtx"
	"github.com/eesrc/horde/pkg/utils/grpcutil"
	"github.com/go-ocf/go-coap"
	"github.com/go-ocf/go-coap/codes"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func coapSend(assert *require.Assertions, msg coap.Message, ep string) coap.Message {
	coapClient := coap.Client{Net: "udp"}
	coapConn, err := coapClient.Dial(ep)
	assert.NoError(err, "CoAP server should listen for requests")
	defer coapConn.Close()
	respMsg, err := coapConn.Exchange(msg)
	assert.NoError(err, "Exchange should be successful")
	return respMsg
}

// Test the CoAP server without any backend server. The server should
// accept request but respond with empty messages to the clients.
func TestCoAPNoServer(t *testing.T) {
	assert := require.New(t)

	port := 2048 + rand.Int31n(4096)
	config := CoAPParameters{
		Endpoint: fmt.Sprintf("127.0.0.1:%d", port),
		Protocol: "udp",
		APNID:    1,
		NASID:    "1",
		AuditLog: true,
	}
	grpcServerEndpoint := fmt.Sprintf("127.0.0.1:%d", 6144+rand.Int31n(1024))
	cc, err := grpcutil.NewGRPCClientConnection(grpcutil.GRPCClientParam{
		ServerEndpoint: grpcServerEndpoint,
	})
	assert.NoError(err, "Client connection should not return an error")
	client := rxtx.NewRxtxClient(cc)

	s := NewCoAPServer(client, config)
	assert.NotNil(s, "Server should be non-nil")
	assert.NoError(s.Start(), "Start() should be successful")
	defer s.Stop()

	assert.Error(s.Start(), "Start() a 2nd time should return error")

	// Do simple CoAP GET, POST, PUT, DELETE to the server. It should return
	// blank responses.

	// GET requests should return a Valid (but empty) response
	msg := coap.NewDgramMessage(coap.MessageParams{
		Type:      coap.Confirmable,
		Code:      codes.GET,
		MessageID: coap.GenerateMessageID(),
	})
	token, err := coap.GenerateToken()
	assert.NoError(err)
	msg.SetToken(token)
	msg.SetPath([]string{"one"})

	respMsg := coapSend(assert, msg, config.Endpoint)
	assert.Equal(codes.Valid, respMsg.Code())

	// POST requests should return a Created response.
	msg = coap.NewDgramMessage(coap.MessageParams{
		Type:      coap.Confirmable,
		Code:      codes.POST,
		MessageID: coap.GenerateMessageID(),
		Payload:   []byte("the POST"),
	})
	token, err = coap.GenerateToken()
	assert.NoError(err)
	msg.SetToken(token)
	msg.SetPath([]string{"one", "two"})
	msg.SetOption(coap.ContentFormat, coap.TextPlain)

	respMsg = coapSend(assert, msg, config.Endpoint)
	assert.Equal(codes.Created, respMsg.Code())

	// PUT request should do the same
	msg = coap.NewDgramMessage(coap.MessageParams{
		Type:      coap.Confirmable,
		Code:      codes.PUT,
		MessageID: coap.GenerateMessageID(),
		Payload:   []byte("the PUT"),
	})
	token, err = coap.GenerateToken()
	assert.NoError(err)
	msg.SetToken(token)
	msg.SetPath([]string{"one", "two", "three"})
	msg.SetOption(coap.ContentFormat, coap.TextPlain)

	respMsg = coapSend(assert, msg, config.Endpoint)
	assert.Equal(codes.Created, respMsg.Code())

	// DELETE requests (which will be ignored since they make no sense)
	// will still return a Delete response
	msg = coap.NewDgramMessage(coap.MessageParams{
		Type:      coap.Confirmable,
		Code:      codes.DELETE,
		MessageID: coap.GenerateMessageID(),
	})
	token, err = coap.GenerateToken()
	assert.NoError(err)
	msg.SetToken(token)
	msg.SetPath([]string{"one", "two", "three", "four"})

	respMsg = coapSend(assert, msg, config.Endpoint)
	assert.Equal(codes.Deleted, respMsg.Code())

	// If we bring up the dummy server it should receive upstream messages matching what we've put in
	dummyServer := newRxtxDummyServer()
	svr, err := grpcutil.NewGRPCServer(grpcutil.GRPCServerParam{
		Endpoint: grpcServerEndpoint,
	})
	assert.NoError(err, "Should not get error when launching grpc server")
	assert.NoError(svr.Launch(func(s *grpc.Server) {
		rxtx.RegisterRxtxServer(s, dummyServer)
	}, 500*time.Millisecond), "Should not get an error when launching the grpc server")
	defer svr.Stop()

	// Wait for the messages from the backlog is received. Should get all four
	// of them.
	assert.True(dummyServer.WaitForMessages(4, 5*time.Second))

	t.Logf("gRPC service runs @ port %s", grpcServerEndpoint)
}

// Test push for clients - create a client to feed messages, then accept
// them on the (device) side.
func TestCoAPPushMessages(t *testing.T) {
	logging.SetLogLevel(logging.DebugLevel)
	assert := require.New(t)

	port := 2048 + rand.Int31n(4096)
	config := CoAPParameters{
		Endpoint: fmt.Sprintf("127.0.0.1:%d", port),
		Protocol: "udp",
		APNID:    1,
		NASID:    "1",
		AuditLog: true,
	}
	grpcServerEndpoint := fmt.Sprintf("127.0.0.1:%d", 6144+rand.Int31n(1024))
	cc, err := grpcutil.NewGRPCClientConnection(grpcutil.GRPCClientParam{
		ServerEndpoint: grpcServerEndpoint,
	})
	assert.NoError(err, "Client connection should not return an error")
	client := rxtx.NewRxtxClient(cc)

	dummyServer := newRxtxDummyServer()
	svr, err := grpcutil.NewGRPCServer(grpcutil.GRPCServerParam{
		Endpoint: grpcServerEndpoint,
	})
	assert.NoError(err, "Should not get error when launching grpc server")
	assert.NoError(svr.Launch(func(s *grpc.Server) {
		rxtx.RegisterRxtxServer(s, dummyServer)
	}, 500*time.Millisecond), "Should not get an error when launching the grpc server")
	defer svr.Stop()

	s := NewCoAPServer(client, config)
	assert.NotNil(s, "Server should be non-nil")
	assert.NoError(s.Start(), "Start() should be successful")
	defer s.Stop()

	// Start the device server
	clientPort := 8192 + rand.Int31n(1024)
	assert.NoError(startClientListener(assert, fmt.Sprintf("127.0.0.1:%d", clientPort)))

	// Push GET, POST, DELETE, PUT to the device server

	// Do a GET
	dummyServer.send(rxtx.DownstreamResponse{
		Msg: &rxtx.Message{
			Id:            1,
			Type:          rxtx.MessageType_CoAPPush,
			Timestamp:     time.Now().UnixNano(),
			RemoteAddress: net.ParseIP("127.0.0.1"),
			RemotePort:    int32(clientPort),
			Coap: &rxtx.CoAPOptions{
				Token:          1,
				TimeoutSeconds: 1,
				Code:           int32(codes.GET),
				Path:           "/get",
				Accept:         int32(coap.AppLwm2mJSON),
				UriQuery:       []string{"foo", "bar"},
			},
		},
	})

	// The server should push the message through a PutMessage request to the
	// server. It should say "Valid" and "GET" in the payload.
	resp := dummyServer.receive(1 * time.Second)
	assert.NotNil(resp)
	assert.Equal(int32(codes.Valid), resp.Msg.Coap.Code)
	assert.Equal("GET", string(resp.Msg.Payload))

	// POST...
	dummyServer.send(rxtx.DownstreamResponse{
		Msg: &rxtx.Message{
			Id:            1,
			Type:          rxtx.MessageType_CoAPPush,
			Timestamp:     time.Now().UnixNano(),
			RemoteAddress: net.ParseIP("127.0.0.1"),
			RemotePort:    int32(clientPort),
			Payload:       []byte("Hello there"),
			Coap: &rxtx.CoAPOptions{
				Token:          1,
				TimeoutSeconds: 1,
				Code:           int32(codes.POST),
				Path:           "/post",
				Accept:         int32(coap.AppLwm2mJSON),
				ContentFormat:  int32(coap.AppLwm2mTLV),
				UriQuery:       []string{"foo", "bar"},
			},
		},
	})

	// The server should push the message through a PutMessage request to the
	// server. It should say "Created" and "POST" in the payload.
	resp = dummyServer.receive(1 * time.Second)
	assert.NotNil(resp)
	assert.Equal(int32(codes.Created), resp.Msg.Coap.Code)
	assert.Equal("POST", string(resp.Msg.Payload))

	// DELETE...
	dummyServer.send(rxtx.DownstreamResponse{
		Msg: &rxtx.Message{
			Id:            1,
			Type:          rxtx.MessageType_CoAPPush,
			Timestamp:     time.Now().UnixNano(),
			RemoteAddress: net.ParseIP("127.0.0.1"),
			RemotePort:    int32(clientPort),
			//			Payload:       []byte("Hello there"),
			Coap: &rxtx.CoAPOptions{
				Token:          1,
				TimeoutSeconds: 1,
				Code:           int32(codes.DELETE),
				Path:           "/delete",
				Accept:         int32(coap.AppLwm2mJSON),
				ContentFormat:  int32(coap.AppLwm2mTLV),
				UriQuery:       []string{"foo", "bar"},
			},
		},
	})

	resp = dummyServer.receive(1 * time.Second)
	assert.NotNil(resp)
	assert.Equal(int32(codes.Deleted), resp.Msg.Coap.Code)
	assert.Equal("DELETE", string(resp.Msg.Payload))
}

func startClientListener(assert *require.Assertions, ep string) error {
	mux := coap.NewServeMux()
	mux.DefaultHandleFunc(func(w coap.ResponseWriter, r *coap.Request) {
		assert.Fail("Did not expect that request")
	})
	mux.HandleFunc("/get", func(w coap.ResponseWriter, r *coap.Request) {
		w.SetCode(codes.Valid)
		w.Write([]byte("GET"))
	})
	mux.HandleFunc("/put", func(w coap.ResponseWriter, r *coap.Request) {
		w.SetCode(codes.Changed)
		w.Write([]byte("PUT"))
	})
	mux.HandleFunc("/post", func(w coap.ResponseWriter, r *coap.Request) {
		w.SetCode(codes.Created)
		w.Write([]byte("POST"))
	})
	mux.HandleFunc("/delete", func(w coap.ResponseWriter, r *coap.Request) {
		w.SetCode(codes.Deleted)
		w.Write([]byte("DELETE"))
	})
	errCh := make(chan error)
	go func(ch chan error) {
		if err := coap.ListenAndServe("udp", ep, mux, nil); err != nil {
			ch <- err
		}
	}(errCh)
	select {
	case e := <-errCh:
		return e
	case <-time.After(250 * time.Millisecond):
		// ok
	}
	return nil
}
