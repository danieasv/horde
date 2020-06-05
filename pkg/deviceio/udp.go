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
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/eesrc/horde/pkg/deviceio/rxtx"

	"github.com/ExploratoryEngineering/logging"
)

type downstreamData struct {
	MessageID          int64
	DestinationAddress net.IP
	DestinationPort    int
	Payload            []byte
}

// UDPListener is used to listen for UDP packets from devices and to send
// UDP packets back. It is supposed to run close to the APN IPSec tunnel (or
// equivalent). The listeners might have different routes set up so they work
// like proxies into Horde proper.
type UDPListener struct {
	downstream    chan downstreamData
	upstream      chan upstreamData
	backlog       chan upstreamData
	terminate     *int32
	listenerWg    *sync.WaitGroup
	client        rxtx.RxtxClient
	backlogger    messageBacklog
	lastMessageID *int64
	config        UDPParameters
	inError       *int32
}

// NewUDPListener creates a new UDP listener instance.
func NewUDPListener(rxtxClient rxtx.RxtxClient, config UDPParameters) *UDPListener {

	ret := &UDPListener{
		listenerWg:    &sync.WaitGroup{},
		client:        rxtxClient,
		lastMessageID: new(int64),
		terminate:     nil,
		config:        config,
		inError:       new(int32),
	}
	atomic.StoreInt64(ret.lastMessageID, 0)
	atomic.StoreInt32(ret.inError, 0)
	return ret
}

// Start starts the UDP listener
func (ut *UDPListener) Start() error {
	if ut.terminate != nil {
		return errors.New("already launched")
	}
	ports, err := ut.config.PortList()
	if err != nil {
		return err
	}
	ut.upstream = make(chan upstreamData)
	ut.downstream = make(chan downstreamData)
	ut.terminate = new(int32)
	atomic.StoreInt32(ut.terminate, 0)
	for _, v := range ports {
		ut.listenerWg.Add(1)
		go ut.listenAndSendOnPort(ut.config.ListenAddress, v)
	}

	ut.backlog = make(chan upstreamData)
	ut.backlogger, err = newMessageBacklog(backlogUDPDatabase)
	if err != nil {
		return err
	}
	if err := ut.backlogger.Reset(); err != nil {
		return err
	}

	go ut.downstreamFeeder()
	go ut.backlogFeeder()
	go ut.mainLoop()
	return nil
}

func (ut *UDPListener) terminated() bool {
	return atomic.LoadInt32(ut.terminate) == 1
}

// Stop stops the listeners.
func (ut *UDPListener) Stop() {
	atomic.StoreInt32(ut.terminate, 1)
	ut.listenerWg.Wait()
	close(ut.upstream)
	close(ut.downstream)
	close(ut.backlog)
}

func (ut *UDPListener) origin() *rxtx.Origin {
	var naslist []int32
	for _, v := range ut.config.NASList() {
		naslist = append(naslist, int32(v))
	}
	return &rxtx.Origin{
		ApnId: int32(ut.config.APNID),
		NasId: naslist,
	}
}

// Feed the downstream channel with items from the upstream service.
func (ut *UDPListener) downstreamFeeder() {
	defer func() {
		if err := recover(); err != nil {
			logging.Warning("Recovered from panic: %v", err)
		}
	}()
	for {
		if ut.terminated() {
			return
		}
		downstreamData, err := ut.getDataWithRetry()
		if err != nil {
			time.Sleep(sleepOnError)
			continue
		}
		if downstreamData == nil {
			time.Sleep(sleepOnEmpty)
			continue
		}
		ut.downstream <- *downstreamData
	}
}

// Feed the backlog channel with items from the backlog.
func (ut *UDPListener) backlogFeeder() {
	for {
		m := ut.backlogger.Get(true)
		if ut.terminated() {
			return
		}
		if m != nil {
			ut.backlog <- *m
		}
	}
}

// sendUPstream sends data to the upstream service and handles
// any return values.
func (ut *UDPListener) sendUpstream(msg *upstreamData) error {
	resp, err := ut.sendUpstreamWithRetry(msg)
	if err != nil {
		ut.backlogger.CancelRemove(msg)
		return err
	}
	ut.backlogger.ConfirmRemove(msg)
	if resp == nil || resp.Msg == nil {
		return nil
	}
	if len(resp.Msg.Payload) != 0 {
		ut.sendDownstream(&downstreamData{
			MessageID:          resp.Msg.Id,
			DestinationAddress: net.IP(resp.Msg.RemoteAddress),
			DestinationPort:    int(resp.Msg.RemotePort),
			Payload:            resp.Msg.Payload,
		}, msg.Conn)
	}
	return nil
}

// sendDownstream sends data downstream and send an ack afterwards.
func (ut *UDPListener) sendDownstream(msg *downstreamData, conn *net.UDPConn) {
	ra, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", msg.DestinationAddress.String(), msg.DestinationPort))
	if err != nil {
		sendAckWithRetry(ut.client, msg.MessageID, rxtx.ErrorCode_NETWORK)
		return
	}
	atomic.StoreInt64(ut.lastMessageID, msg.MessageID)
	n, _, err := conn.WriteMsgUDP(msg.Payload, nil, ra)
	if err != nil {
		logging.Warning("Got error when sending %d bytes to %s:%d: %v", len(msg.Payload), msg.DestinationAddress, msg.DestinationPort, err)
		sendAckWithRetry(ut.client, msg.MessageID, rxtx.ErrorCode_NETWORK)
		return
	}

	if ut.config.AuditLog {
		logging.Info("Sent %d bytes to %s", len(msg.Payload), ra.String())
	}
	// Update the message ID. If it fails after this we won't retry
	if n != len(msg.Payload) {
		logging.Warning("Wanted to send %d bytes but only sent %d", n, len(msg.Payload))
		sendAckWithRetry(ut.client, msg.MessageID, rxtx.ErrorCode_TOO_LARGE)
		return
	}
	sendAckWithRetry(ut.client, msg.MessageID, rxtx.ErrorCode_SUCCESS)
}

func (ut *UDPListener) listenAndSendOnPort(listenAddress string, port int) {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", listenAddress, port))
	defer ut.listenerWg.Done()
	if err != nil {
		logging.Error("Can't resolve address %s:%d: %v. No listener launched for port %d.", listenAddress, port, err, port)
		return
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		logging.Error("Could not listen on %s:%d: %v. No listener launched for port %d", listenAddress, port, err, port)
		return
	}
	defer conn.Close()

	go func(conn *net.UDPConn) {
		for {
			if ut.terminated() {
				return
			}
			select {
			case msg, ok := <-ut.downstream:
				if !ok {
					return
				}
				ut.sendDownstream(&msg, conn)
			case <-time.After(1000 * time.Millisecond):
				continue
			}
		}
	}(conn)
	logging.Info("Listening for UDP packets on %s", addr.String())
	for {
		if ut.terminated() {
			return
		}
		conn.SetReadDeadline(time.Now().Add(1000 * time.Millisecond))
		buf := make([]byte, 2048)
		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}
		if n == 0 {
			logging.Warning("Got 0 byte payload from %s", addr.String())
			continue
		}

		ut.upstream <- upstreamData{
			Msg: rxtx.Message{
				Type:          rxtx.MessageType_UDP,
				Timestamp:     time.Now().UnixNano(),
				Payload:       buf[0:n],
				LocalPort:     int32(port),
				RemoteAddress: addr.IP,
				RemotePort:    int32(addr.Port),
			},
			Conn: conn,
		}
		if ut.config.AuditLog {
			logging.Info("Received %d bytes from %s", n, addr.String())
		}
	}
}

func (ut *UDPListener) mainLoop() {
	for {
		if ut.terminated() {
			return
		}
		select {
		case msg, ok := <-ut.upstream:
			if !ok {
				return
			}
			if err := ut.backlogger.Add(&msg); err != nil {
				logging.Error("Got error adding message to backlog: %v", err)
			}
			ut.sendUpstream(&msg)
		case msg, ok := <-ut.backlog:
			if !ok {
				return
			}
			if err := ut.sendUpstream(&msg); err != nil {
				ut.backlogger.CancelRemove(&msg)
				continue
			}
			ut.backlogger.ConfirmRemove(&msg)
		}
	}
}

// Do a gRPC get data call with retry
func (ut *UDPListener) getDataWithRetry() (*downstreamData, error) {
	ctx, done := context.WithTimeout(context.Background(), grpcTimeout)
	defer done()

	req := &rxtx.DownstreamRequest{
		Origin: ut.origin(),
		Type:   rxtx.MessageType_UDP,
	}

	res, err := ut.client.GetMessage(ctx, req)
	if err != nil {
		ctx2, done2 := context.WithTimeout(context.Background(), grpcRetryTimeout)
		defer done2()
		res, err = ut.client.GetMessage(ctx2, req)
	}

	if err != nil {
		if atomic.LoadInt32(ut.inError) == 0 {
			// Log only the first time the request failed
			logging.Error("GetData request failed with retry: %v", err)
			atomic.StoreInt32(ut.inError, 1)
		}
		return nil, err
	}
	if atomic.LoadInt32(ut.inError) == 1 {
		atomic.StoreInt32(ut.inError, 0)
		logging.Info("GetData request has recovered")
	}
	if res == nil || res.Msg == nil {
		return nil, nil
	}
	if res.Msg.Id == 0 {
		panic("Message ID is 0")
	}

	if len(res.Msg.Payload) == 0 {
		logging.Warning("Got message ID but no payload: %+v", res)
		return nil, nil
	}

	return &downstreamData{
		MessageID:          res.Msg.Id,
		DestinationAddress: net.IP(res.Msg.RemoteAddress),
		DestinationPort:    int(res.Msg.RemotePort),
		Payload:            res.Msg.Payload,
	}, nil
}

// sendUpstreamWithRetry sends data to the upstream service with an optional
// retry. If retry fails it will return an error.
func (ut *UDPListener) sendUpstreamWithRetry(msg *upstreamData) (*rxtx.DownstreamResponse, error) {
	ctx, done := context.WithTimeout(context.Background(), grpcTimeout)
	defer done()

	req := &rxtx.UpstreamRequest{
		Origin:           ut.origin(),
		ExpectDownstream: true,
		Msg:              &msg.Msg,
	}
	res, err := ut.client.PutMessage(ctx, req)
	if err != nil {
		ctx2, done2 := context.WithTimeout(context.Background(), grpcRetryTimeout)
		defer done2()
		res, err = ut.client.PutMessage(ctx2, req)
	}
	return res, err
}
