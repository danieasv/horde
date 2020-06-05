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
	"errors"
	"net"
	"os"
	"sync"
	"testing"

	"github.com/eesrc/horde/pkg/deviceio/rxtx"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func sendUDP(endpoint string, buf []byte) error {
	conn, err := net.Dial("udp", endpoint)
	if err != nil {
		return err
	}
	defer conn.Close()
	_, err = conn.Write(buf)
	return err
}

func TestUDPListener(t *testing.T) {
	assert := require.New(t)
	defer os.Remove(backlogUDPDatabase)
	wg := &sync.WaitGroup{}
	wg.Add(4)

	client := &udpClient{wg: wg}

	// Create config with duplicate ports. The net result is that we'll have
	// a server listening on four ports. The duplicates will fail.
	config := UDPParameters{
		Ports:         "9000,9001,9002,9003,9001,9002,9003",
		ListenAddress: "127.0.0.1",
		APNID:         1,
		NASID:         "1",
		AuditLog:      true,
	}

	l := NewUDPListener(client, config)
	assert.NotNil(l)

	assert.NoError(l.Start())

	assert.NoError(sendUDP("127.0.0.1:9000", []byte("hello")))
	assert.NoError(sendUDP("127.0.0.1:9001", []byte("hello")))
	assert.NoError(sendUDP("127.0.0.1:9002", []byte("hello")))
	assert.NoError(sendUDP("127.0.0.1:9003", []byte("hello")))

	wg.Wait()

	l.Stop()
}

// Stub the client. This one returns errors every other time it's called.
type udpClient struct {
	wg         *sync.WaitGroup
	getcount   int
	putcount   int
	ackcount   int
	udpOptions *rxtx.UDPOptions
}

func (c *udpClient) GetMessage(ctx context.Context, in *rxtx.DownstreamRequest, opts ...grpc.CallOption) (*rxtx.DownstreamResponse, error) {
	if c.getcount%2 == 0 {
		c.getcount++
		return nil, errors.New("error")
	}
	c.getcount++
	return &rxtx.DownstreamResponse{
		Msg: &rxtx.Message{
			Id:            int64(c.getcount),
			Type:          rxtx.MessageType_UDP,
			RemoteAddress: net.ParseIP("127.0.0.1"),
			RemotePort:    4711,
			Payload:       make([]byte, 1024),
			Udp:           c.udpOptions,
		},
	}, nil
}

func (c *udpClient) PutMessage(ctx context.Context, in *rxtx.UpstreamRequest, opts ...grpc.CallOption) (*rxtx.DownstreamResponse, error) {
	if c.putcount%2 == 0 {
		c.putcount++
		return nil, errors.New("error")
	}
	c.putcount++
	c.wg.Done()
	return &rxtx.DownstreamResponse{
		Msg: &rxtx.Message{
			Id:            int64(c.getcount),
			Type:          rxtx.MessageType_UDP,
			RemoteAddress: net.ParseIP("127.0.0.1"),
			RemotePort:    4711,
			Payload:       make([]byte, 1024),
			Udp:           c.udpOptions,
		},
	}, nil
}

func (c *udpClient) Ack(ctx context.Context, in *rxtx.AckRequest, opts ...grpc.CallOption) (*rxtx.AckResponse, error) {
	if c.ackcount%2 == 0 {
		c.ackcount++
		return nil, errors.New("error")
	}
	c.ackcount++
	return &rxtx.AckResponse{}, nil
}

func TestCIDR(t *testing.T) {
	assert := require.New(t)

	_, net1, err := net.ParseCIDR("10.0.0.1/13")
	assert.NoError(err)

	assert.True(net1.Contains(net.ParseIP("10.1.0.1")))
	assert.True(net1.Contains(net.ParseIP("10.7.0.254")))
	assert.False(net1.Contains(net.ParseIP("10.8.0.1")))

	_, net2, err := net.ParseCIDR("10.8.0.1/13")
	assert.NoError(err)

	assert.True(net2.Contains(net.ParseIP("10.8.0.1")))
	assert.True(net2.Contains(net.ParseIP("10.15.0.254")))
	assert.False(net2.Contains(net.ParseIP("10.1.0.1")))
}
