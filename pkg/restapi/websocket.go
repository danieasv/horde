package restapi

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
	"errors"
	"net/http"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"

	"github.com/ExploratoryEngineering/logging"
	"github.com/eesrc/horde/pkg/api/apipb"
	"github.com/eesrc/horde/pkg/api/apitoolbox"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/gorilla/websocket"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const handshakeTimeout = 60 * time.Second
const keepAliveTimeout = 30 * time.Second

var upgrader = websocket.Upgrader{
	ReadBufferSize:    4096,
	WriteBufferSize:   4096,
	EnableCompression: true,
	HandshakeTimeout:  handshakeTimeout,

	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type streamServerStub struct {
	ctx     context.Context
	msgChan chan *apipb.OutputDataMessage
}

func (s *streamServerStub) Context() context.Context {
	return s.ctx
}

func (s *streamServerStub) Send(m *apipb.OutputDataMessage) error {
	logging.Debug("Sending data (%+v)", m)
	s.msgChan <- m
	return nil
}

func (s *streamServerStub) Recv() *apipb.OutputDataMessage {
	ret, ok := <-s.msgChan
	if !ok {
		return nil
	}
	return ret
}

// This is the actual client implementation that will be used.
func (s *streamServerStub) StubRecv() (*apipb.OutputDataMessage, error) {
	ret, ok := <-s.msgChan
	if !ok {
		return nil, errors.New("eof")
	}
	return ret, nil
}
func (s *streamServerStub) CloseSend() error {
	return nil
}

// End stubs

func (s *streamServerStub) SetHeader(metadata.MD) error {
	return nil
}

func (s *streamServerStub) SendHeader(metadata.MD) error {
	return nil
}

func (s *streamServerStub) SetTrailer(metadata.MD) {

}

func (s *streamServerStub) SendMsg(m interface{}) error {
	s.msgChan <- m.(*apipb.OutputDataMessage)
	return nil
}

func (s *streamServerStub) RecvMsg(m interface{}) error {
	ret, ok := <-s.msgChan
	if !ok {
		return errors.New("eof")
	}
	tmp := m.(*apipb.OutputDataMessage)
	*tmp = *ret
	return nil
}

func newWebsocketHandler(collectionID, deviceID string, apiService apipb.HordeServer) http.HandlerFunc {
	if apiService == nil {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Nothing to see here!"))
		}
	}
	return func(w http.ResponseWriter, r *http.Request) {
		req := &apipb.MessageStreamRequest{
			CollectionId: &wrappers.StringValue{Value: collectionID},
		}
		if deviceID != "" {
			req.DeviceId = &wrappers.StringValue{Value: deviceID}
		}

		// The proper client will use the gRPC client which is different. This
		// is just a workaround until the gRPC server is finished.
		stream := &streamServerStub{
			ctx:     r.Context(),
			msgChan: make(chan *apipb.OutputDataMessage),
		}
		err := apiService.MessageStream(req, stream)
		if err != nil {
			reportError(w, runtime.HTTPStatusFromCode(status.Code(err)), err.Error(), nil)
			return
		}

		// Future bugs:
		// Bug: The first read from the stream will return the error from the
		// service. Attempt a first read before upgrading the websocket to
		// ensure the error is sent as early as possible.
		//
		// Bug II: Move the context data into the metadata header of the
		// call to ensure proper authentication. The current context abuse isn't
		// carried over via the gRPC client context.

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logging.Warning("Error upgrading web socket: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer conn.Close()
		cc := make(chan bool)
		go func(c *websocket.Conn) {
			for {
				if _, _, err := c.NextReader(); err != nil {
					c.Close()
					cc <- true
					return
				}
			}
		}(conn)
		messageChan := make(chan *apipb.OutputDataMessage)
		go func() {
			defer close(messageChan)
			for {
				msg, err := stream.StubRecv()
				if err != nil {
					conn.WriteControl(websocket.CloseMessage, []byte{}, time.Now().Add(keepAliveTimeout))
					return
				}
				select {
				case <-cc:
					if err := stream.CloseSend(); err != nil {
						logging.Warning("Got error closing server stream: %v", err)
						return
					}
				default:
					// Keep on truckin'
				}
				messageChan <- msg
			}
		}()

		m := apitoolbox.JSONMarshaler()
		for {
			select {
			case <-cc:
				stream.CloseSend()
				return
			case msg := <-messageChan:
				str, err := m.MarshalToString(msg)
				if err != nil {
					logging.Warning("Error marshaling JSON message: %v", err)
					continue
				}
				conn.SetWriteDeadline(time.Now().Add(keepAliveTimeout))
				if err := conn.WriteMessage(websocket.TextMessage, []byte(str)); err != nil {
					logging.Info("Error writing websocket message. Exiting loop: %v", err)
					conn.Close()
					return
				}
				// Got some message - send it
			case <-time.After(keepAliveTimeout):
				// send KeepAlive-message
				msg := &apipb.OutputDataMessage{
					Type: apipb.OutputDataMessage_keepalive,
				}
				str, err := m.MarshalToString(msg)
				if err != nil {
					logging.Warning("Error marshaling JSON message: %v", err)
					continue
				}
				conn.SetWriteDeadline(time.Now().Add(keepAliveTimeout))
				if err := conn.WriteMessage(websocket.TextMessage, []byte(str)); err != nil {
					return
				}
			}
		}
	}
}
