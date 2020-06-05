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
	"sync/atomic"
	"time"

	"github.com/ExploratoryEngineering/logging"
	"github.com/eesrc/horde/pkg/deviceio/rxtx"
)

func newRxtxDummyServer() *rxtxDummyServer {
	ret := &rxtxDummyServer{
		received:   new(int32),
		downstream: make(chan rxtx.DownstreamResponse),
		upstream:   make(chan rxtx.UpstreamRequest),
	}
	atomic.StoreInt32(ret.received, 0)
	return ret
}

type rxtxDummyServer struct {
	received   *int32
	downstream chan rxtx.DownstreamResponse
	upstream   chan rxtx.UpstreamRequest
}

// WaitForMessages waits for count messages up to timeout
func (r *rxtxDummyServer) WaitForMessages(count int, timeout time.Duration) bool {
	for i := 0; i < 10; i++ {
		time.Sleep(timeout / 10)
		r := atomic.LoadInt32(r.received)
		if int(r) >= count {
			return true
		}
		logging.Debug("Waiting %d of 10: %d received", i, r)
	}
	return false
}

func (r *rxtxDummyServer) GetMessage(ctx context.Context, req *rxtx.DownstreamRequest) (*rxtx.DownstreamResponse, error) {
	select {
	case resp := <-r.downstream:
		return &resp, nil
	default:
		return &rxtx.DownstreamResponse{}, nil
	}
}

func (r *rxtxDummyServer) PutMessage(ctx context.Context, req *rxtx.UpstreamRequest) (*rxtx.DownstreamResponse, error) {
	atomic.AddInt32(r.received, 1)
	select {
	case r.upstream <- *req:
	case <-time.After(5 * time.Millisecond):
		logging.Debug("Could not forward message on channel")
	}
	return &rxtx.DownstreamResponse{}, nil
}

func (r *rxtxDummyServer) Ack(ctx context.Context, req *rxtx.AckRequest) (*rxtx.AckResponse, error) {
	return &rxtx.AckResponse{}, nil
}

func (r *rxtxDummyServer) send(msg rxtx.DownstreamResponse) {
	r.downstream <- msg
}

func (r *rxtxDummyServer) receive(timeout time.Duration) *rxtx.UpstreamRequest {
	select {
	case ret := <-r.upstream:
		return &ret
	case <-time.After(timeout):
		return nil
	}
}
