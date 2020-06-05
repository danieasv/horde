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
	"database/sql"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/eesrc/horde/pkg/deviceio/rxtx"
	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
	"github.com/eesrc/horde/pkg/storage/sqlstore"
	"github.com/eesrc/horde/pkg/storage/storetest"
	"github.com/go-ocf/go-coap"
	"github.com/go-ocf/go-coap/codes"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"

	// SQLite3 driver for testing, local instances and in-memory database
	_ "github.com/mattn/go-sqlite3"
)

// The in memory store is kept open between tests so this purges the message
// table for each test
func purgeMessages() {
	db, _ := sql.Open("sqlite3", memoryDB)
	db.Exec("drop table downstream_messages")
}

const memoryDB = "file::memory:?cache=shared"

func TestRxTxReceiverUDP(t *testing.T) {
	defer purgeMessages()
	assert := require.New(t)
	publisher := make(chan model.DataMessage, 20)

	apnStore := sqlstore.NewMemoryAPNStore()
	assert.NoError(apnStore.CreateAPN(model.APN{ID: 1, Name: "test.apn"}))
	assert.NoError(apnStore.CreateNAS(model.NAS{ID: 1, CIDR: "10.0.0.0/16", Identifier: "NAS01", ApnID: 1}))
	assert.NoError(apnStore.CreateAPN(model.APN{ID: 2, Name: "test2.apn"}))
	assert.NoError(apnStore.CreateNAS(model.NAS{ID: 2, CIDR: "10.1.0.0/16", Identifier: "NAS02", ApnID: 2}))

	apnConfig, err := storage.NewAPNCache(apnStore)
	assert.NoError(err)

	datastore := sqlstore.NewMemoryStore()
	te := storetest.NewTestEnvironment(t, datastore)
	d := model.NewDevice()
	d.ID = datastore.NewDeviceID()
	d.IMSI = 1001
	d.IMEI = 1001
	d.CollectionID = te.C1.ID
	assert.NoError(datastore.CreateDevice(te.U1.ID, d))

	assert.NoError(apnStore.CreateAllocation(model.Allocation{
		IP:      net.ParseIP("10.0.0.1"),
		IMSI:    1001,
		IMEI:    1001,
		ApnID:   1,
		NasID:   1,
		Created: time.Now(),
	}))
	assert.NoError(apnStore.CreateAllocation(model.Allocation{
		IP:      net.ParseIP("10.1.0.1"),
		IMSI:    1001,
		IMEI:    1001,
		ApnID:   2,
		NasID:   2,
		Created: time.Now(),
	}))
	// This is a device that has just an allocation
	assert.NoError(apnStore.CreateAllocation(model.Allocation{
		IP:      net.ParseIP("10.0.0.2"),
		IMSI:    2001,
		IMEI:    2001,
		ApnID:   1,
		NasID:   1,
		Created: time.Now(),
	}))
	params := sqlstore.Parameters{
		ConnectionString: memoryDB,
		Type:             "sqlite3",
		CreateSchema:     true,
	}
	downstreamStore, err := sqlstore.NewDownstreamStore(datastore, params, 1, 1)
	assert.NoError(err)

	r := NewRxTxReceiver(apnConfig, datastore, apnStore, downstreamStore, publisher)
	assert.NotNil(r)

	// Empty request
	_, err = r.PutMessage(context.Background(), &rxtx.UpstreamRequest{})
	assert.Error(err)

	// No remote address
	_, err = r.PutMessage(context.Background(), &rxtx.UpstreamRequest{
		Msg: &rxtx.Message{},
	})
	assert.Error(err)

	// No origin but msg + remote address
	_, err = r.PutMessage(context.Background(), &rxtx.UpstreamRequest{
		Msg: &rxtx.Message{RemoteAddress: net.ParseIP("1.2.3.4")},
	})
	assert.Error(err)

	// Unknown APN
	_, err = r.PutMessage(context.Background(), &rxtx.UpstreamRequest{
		Origin:     &rxtx.Origin{ApnId: 10, NasId: []int32{1}},
		Redelivery: false,
		Msg: &rxtx.Message{
			RemoteAddress: net.ParseIP("10.0.0.1"),
			Payload:       nil,
		},
	})
	assert.Error(err)

	// No message field
	_, err = r.PutMessage(context.Background(), &rxtx.UpstreamRequest{
		Origin: &rxtx.Origin{ApnId: 0, NasId: []int32{0}},
	})
	assert.Error(err)

	// Unknown device, known APN
	_, err = r.PutMessage(context.Background(), &rxtx.UpstreamRequest{
		Origin:     &rxtx.Origin{ApnId: 1, NasId: []int32{1}},
		Redelivery: false,
		Msg: &rxtx.Message{
			RemoteAddress: net.ParseIP("10.0.0.4"),
			Payload:       nil,
		},
	})
	// Unknown device is "yeah ok whatever"
	assert.NoError(err)

	// Successful send
	genericMsg := rxtx.Message{
		Id:            1,
		Type:          rxtx.MessageType_UDP,
		Timestamp:     time.Now().UnixNano(),
		RemoteAddress: net.ParseIP("10.0.0.1"),
		RemotePort:    4711,
		LocalPort:     4712,
		Udp:           &rxtx.UDPOptions{},
		Payload:       []byte("Hello there"),
	}
	_, err = r.PutMessage(context.Background(), &rxtx.UpstreamRequest{
		Origin:     &rxtx.Origin{ApnId: 1, NasId: []int32{1}},
		Redelivery: false,
		Msg:        &genericMsg,
	})
	assert.NoError(err)

	ensureReceive := func() {
		select {
		case d := <-publisher:
			assert.Equal(d.Device.IMSI, int64(1001))
			assert.Equal(d.Transport, model.UDPTransport)
		default:
			assert.Fail("Did not receive a message")
		}
	}
	ensureReceive()

	// Repeat but use a mis-matched IP address. This *can* happen if the routing
	// is broken or one of the listeners are misconfigured. We have found the
	// device so it's an existing one but the NAS parameter is incorrect.
	// Also mark as redelivery. The response should be empty.
	ipMsg := genericMsg
	ipMsg.RemoteAddress = net.ParseIP("10.1.0.1")
	res, err := r.PutMessage(context.Background(), &rxtx.UpstreamRequest{
		Origin:     &rxtx.Origin{ApnId: 2, NasId: []int32{4}},
		Redelivery: true,
		Msg:        &ipMsg,
	})
	assert.NoError(err)
	assert.Nil(res.Msg)

	ensureReceive()

	// Stale device data - the allocation exists but the device doesn't
	staleMsg := genericMsg
	staleMsg.RemoteAddress = net.ParseIP("10.0.0.2")
	_, err = r.PutMessage(context.Background(), &rxtx.UpstreamRequest{
		Origin:     &rxtx.Origin{ApnId: 1, NasId: []int32{1}},
		Redelivery: false,
		Msg:        &staleMsg,
	})
	assert.NoError(err)

	// add downstream message and ensure it is returned when the device
	// pokes. The port must be 0 or the same as the sender port.

	// This should be returned
	msg := &rxtx.Message{
		Id:         100,
		RemotePort: 0,
		Payload:    []byte("upstream"),
		Udp:        &rxtx.UDPOptions{},
		Type:       rxtx.MessageType_UDPPull,
	}
	buf, err := proto.Marshal(msg)
	assert.NoError(err)
	assert.NoError(downstreamStore.Create(1, 1, d.ID, 100, model.UDPPullTransport, buf))

	// This should not be returned since the port is different
	msg.RemotePort = 4712
	msg.Id = 101
	msg.Payload = []byte("other port")
	buf, err = proto.Marshal(msg)
	assert.NoError(err)
	assert.NoError(downstreamStore.Create(1, 1, d.ID, 101, model.UDPTransport, buf))

	// This should not be returned since the transport is different
	msg.RemotePort = 4711
	msg.Id = 102
	msg.Udp = nil
	msg.Coap = &rxtx.CoAPOptions{Path: "/something"}
	msg.Payload = []byte("other transport")
	msg.Type = rxtx.MessageType_CoAPPush
	buf, err = proto.Marshal(msg)
	assert.NoError(err)
	assert.NoError(downstreamStore.Create(1, 1, d.ID, 102, model.CoAPTransport, buf))

	resp, err := r.PutMessage(context.Background(), &rxtx.UpstreamRequest{
		Origin:           &rxtx.Origin{ApnId: 1, NasId: []int32{1}},
		ExpectDownstream: true,
		Redelivery:       false,
		Msg:              &genericMsg,
	})
	assert.NoError(err)
	ensureReceive()

	assert.NotNil(resp)
	assert.NotNil(resp.Msg)
	assert.Equal("upstream", string(resp.Msg.Payload))

	// Ack the message and try again. Nothing should be returned
	_, err = r.Ack(context.Background(), &rxtx.AckRequest{
		MessageId: resp.Msg.Id,
		Result:    rxtx.ErrorCode_SUCCESS,
	})
	assert.NoError(err)

	mid := resp.Msg.Id

	resp, err = r.PutMessage(context.Background(), &rxtx.UpstreamRequest{
		Origin:           &rxtx.Origin{ApnId: 1, NasId: []int32{1}},
		Redelivery:       false,
		ExpectDownstream: true,
		Msg:              &genericMsg,
	})
	assert.NoError(err)
	ensureReceive()
	assert.NotNil(resp)
	assert.Nil(resp.Msg)

	// Ack unknown message
	_, err = r.Ack(context.Background(), &rxtx.AckRequest{})
	assert.NoError(err)

	// Ack same message twice
	_, err = r.Ack(context.Background(), &rxtx.AckRequest{MessageId: mid, Result: rxtx.ErrorCode_SUCCESS})
	assert.NoError(err)

	// ...and with error (it will be ignored)
	_, err = r.Ack(context.Background(), &rxtx.AckRequest{MessageId: mid, Result: rxtx.ErrorCode_NETWORK})
	assert.NoError(err)

	// Drain the message store by requesting the messages we added earlier
	getr, err := r.GetMessage(context.Background(), &rxtx.DownstreamRequest{
		Origin: &rxtx.Origin{ApnId: 1, NasId: []int32{1}},
		Type:   rxtx.MessageType_CoAPPush,
	})
	assert.NoError(err)
	assert.NotNil(getr.Msg)
	assert.Equal(getr.Msg.Id, int64(102))
}

func TestRxTxReceiverCoAP(t *testing.T) {
	defer purgeMessages()

	assert := require.New(t)
	publisher := make(chan model.DataMessage, 20)

	apnStore := sqlstore.NewMemoryAPNStore()
	assert.NoError(apnStore.CreateAPN(model.APN{ID: 1, Name: "test.apn"}))
	assert.NoError(apnStore.CreateNAS(model.NAS{ID: 1, CIDR: "10.0.0.0/16", Identifier: "NAS01", ApnID: 1}))

	apnConfig, err := storage.NewAPNCache(apnStore)
	assert.NoError(err)

	datastore := sqlstore.NewMemoryStore()
	te := storetest.NewTestEnvironment(t, datastore)
	d := model.NewDevice()
	d.ID = datastore.NewDeviceID()
	d.IMSI = 1001
	d.IMEI = 1001
	d.CollectionID = te.C1.ID
	assert.NoError(datastore.CreateDevice(te.U1.ID, d))

	assert.NoError(apnStore.CreateAllocation(model.Allocation{
		IP:      net.ParseIP("10.0.0.1"),
		IMSI:    1001,
		IMEI:    1001,
		ApnID:   1,
		NasID:   1,
		Created: time.Now(),
	}))
	params := sqlstore.Parameters{
		ConnectionString: memoryDB,
		Type:             "sqlite3",
		CreateSchema:     true,
	}
	downstreamStore, err := sqlstore.NewDownstreamStore(datastore, params, 1, 1)
	assert.NoError(err)

	r := NewRxTxReceiver(apnConfig, datastore, apnStore, downstreamStore, publisher)
	assert.NotNil(r)

	genericMsg := rxtx.Message{
		Id:            1,
		Type:          rxtx.MessageType_CoAPUpstream,
		Timestamp:     time.Now().UnixNano(),
		RemoteAddress: net.ParseIP("10.0.0.1"),
		RemotePort:    4711,
		LocalPort:     5683,
		Coap: &rxtx.CoAPOptions{
			Path: "/some/path",
			Code: int32(codes.POST),
		},
		Payload: []byte("Hello there"),
	}

	// Missing coap options should return an error.
	noOptions := genericMsg
	noOptions.Coap = nil
	_, err = r.PutMessage(context.Background(), &rxtx.UpstreamRequest{
		Origin:     &rxtx.Origin{ApnId: 1, NasId: []int32{1}},
		Redelivery: false,
		Msg:        &noOptions,
	})
	assert.Error(err)

	// Now send a CoAP message. It's basically the same as before
	_, err = r.PutMessage(context.Background(), &rxtx.UpstreamRequest{
		Origin:     &rxtx.Origin{ApnId: 1, NasId: []int32{1}},
		Redelivery: false,
		Msg:        &genericMsg,
	})
	assert.NoError(err)

	ensureReceive := func() {
		select {
		case d := <-publisher:
			assert.Equal(d.Device.IMSI, int64(1001))
			assert.Equal(d.Transport, model.CoAPPullTransport)
		default:
			assert.Fail("Did not receive a message")
		}
	}
	ensureReceive()

	// Add handlers and ensure they are sent the right way
	r.AddCoAPHandler("/rd", dummyLwM2Mhandler)
	r.AddCoAPHandler("/u", dummyRegHandler)
	r.AddCoAPHandler("/u", dummyRegHandler) // overwrite is OK but should log error

	pathMsg := genericMsg
	pathMsg.Coap = &rxtx.CoAPOptions{
		Code:          int32(codes.GET),
		Path:          "rd",
		LocationPath:  []string{"loc1", "loc1"},
		UriQuery:      []string{"id=12"},
		Type:          int32(coap.Confirmable),
		ContentFormat: int32(coap.AppLwm2mTLV),
	}
	lwm2mres, err := r.PutMessage(context.Background(), &rxtx.UpstreamRequest{
		Origin:     &rxtx.Origin{ApnId: 1, NasId: []int32{1}},
		Redelivery: false,
		Msg:        &pathMsg,
	})

	assert.NoError(err)
	assert.NotNil(lwm2mres)
	assert.NotNil(lwm2mres.Msg)
	assert.Equal("lwm2mhandler", string(lwm2mres.Msg.Payload))

	// A redelivery message isn't passed on to handlers
	lwm2mres, err = r.PutMessage(context.Background(), &rxtx.UpstreamRequest{
		Origin:     &rxtx.Origin{ApnId: 1, NasId: []int32{1}},
		Redelivery: true,
		Msg:        &pathMsg,
	})

	assert.NoError(err)
	assert.NotNil(lwm2mres)
	assert.Nil(lwm2mres.Msg)

	// Set up downstream message. This won't get delivered until a fresh message
	// is received.

	msg := &rxtx.Message{
		Id:         200,
		Type:       rxtx.MessageType_CoAPPull,
		RemotePort: 0,
		Payload:    []byte("pull"),
		Coap: &rxtx.CoAPOptions{
			Path: "/somepath",
		},
	}
	buf, err := proto.Marshal(msg)
	assert.NoError(err)
	assert.NoError(downstreamStore.Create(1, 1, d.ID, 200, model.CoAPPullTransport, buf))

	msg.Id = 201
	msg.Payload = []byte("push")
	msg.Type = rxtx.MessageType_CoAPPush
	buf, err = proto.Marshal(msg)
	assert.NoError(err)
	assert.NoError(downstreamStore.Create(1, 1, d.ID, 201, model.CoAPTransport, buf))

	// Redelivery messages that doesn't match a path gets forwarded the usual
	// way
	redeliver, err := r.PutMessage(context.Background(), &rxtx.UpstreamRequest{
		Origin:           &rxtx.Origin{ApnId: 1, NasId: []int32{1}},
		Redelivery:       true,
		ExpectDownstream: true,
		Msg:              &genericMsg,
	})
	assert.NoError(err)
	assert.Nil(redeliver.Msg)
	ensureReceive()

	getMsg := genericMsg
	getMsg.Coap.Code = int32(codes.GET)
	// A new fresh message should return the "pull" message above
	pull, err := r.PutMessage(context.Background(), &rxtx.UpstreamRequest{
		Origin:           &rxtx.Origin{ApnId: 1, NasId: []int32{1}},
		Redelivery:       false,
		ExpectDownstream: true,
		Msg:              &getMsg,
	})
	assert.NoError(err)
	assert.NotNil(pull)
	assert.NotNil(pull.Msg)
	assert.Equal(int64(200), pull.Msg.Id)
	assert.Equal("pull", string(pull.Msg.Payload))
	//	ensureReceive()

	_, err = r.Ack(context.Background(), &rxtx.AckRequest{MessageId: pull.Msg.Id, Result: rxtx.ErrorCode_SUCCESS})
	assert.NoError(err)

	// ...the next should not return the push message
	nopush, err := r.PutMessage(context.Background(), &rxtx.UpstreamRequest{
		Origin:           &rxtx.Origin{ApnId: 1, NasId: []int32{1}},
		Redelivery:       false,
		ExpectDownstream: true,
		Msg:              &getMsg,
	})
	assert.NoError(err)
	assert.NotNil(nopush)
	assert.Nil(nopush.Msg)
	// ensureReceive()

	// A GetMessage request should return the first push message above, then
	// the new push message just added
	push, err := r.GetMessage(context.Background(), &rxtx.DownstreamRequest{
		Origin: &rxtx.Origin{ApnId: 1, NasId: []int32{1}},
		Type:   rxtx.MessageType_CoAPPush,
	})
	assert.NoError(err)
	assert.NotNil(push)
	assert.NotNil(push.Msg)
	assert.Equal("push", string(push.Msg.Payload))
}

func dummyRegHandler(apnID int, nasID int, device *model.Device, req *rxtx.UpstreamRequest) (*rxtx.DownstreamResponse, ResponseCallback, error) {
	return &rxtx.DownstreamResponse{Msg: &rxtx.Message{Id: 1, Payload: []byte("reg")}}, nil, nil
}

func dummyLwM2Mhandler(apnID int, nasID int, device *model.Device, req *rxtx.UpstreamRequest) (*rxtx.DownstreamResponse, ResponseCallback, error) {
	return &rxtx.DownstreamResponse{Msg: &rxtx.Message{Id: 1, Payload: []byte("lwm2mhandler")}}, nil, nil
}

func TestSend(t *testing.T) {
	defer purgeMessages()

	assert := require.New(t)

	r, d := setupCoap(assert, t)

	ctx, done := context.WithTimeout(context.Background(), 1*time.Second)
	defer done()

	// nil message
	code, err := r.Send(ctx, d, nil, true)
	assert.Error(err)
	assert.Equal(rxtx.ErrorCode_CLIENT_ERROR, code)

	// Invalid type
	msg := &rxtx.Message{
		Id:   1000,
		Type: rxtx.MessageType_CoAPUpstream,
	}

	code, err = r.Send(ctx, d, msg, true)
	assert.Error(err)
	assert.Equal(rxtx.ErrorCode_CLIENT_ERROR, code)

	// Missing remote address
	msg.Type = rxtx.MessageType_UDP
	code, err = r.Send(ctx, d, msg, true)
	assert.Error(err)
	assert.Equal(rxtx.ErrorCode_CLIENT_ERROR, code)

	// Missing remote port
	msg.RemoteAddress = net.ParseIP("10.0.0.1")
	code, err = r.Send(ctx, d, msg, true)
	assert.Error(err)
	assert.Equal(rxtx.ErrorCode_CLIENT_ERROR, code)

	// Device has no address
	msg.RemotePort = 10000
	code, err = r.Send(ctx, d, msg, true)
	assert.Error(err)
	assert.Equal(rxtx.ErrorCode_CLIENT_ERROR, code)

	d.Network.AllocatedIP = "10.0.0.1"
	d.Network.ApnID = 1
	d.Network.NasID = 1

	// UDP packet has no payload
	code, err = r.Send(ctx, d, msg, true)
	assert.Error(err)
	assert.Equal(rxtx.ErrorCode_CLIENT_ERROR, code)

	// Coap type with no options
	msg.Type = rxtx.MessageType_CoAPPull
	code, err = r.Send(ctx, d, msg, true)
	assert.Error(err)
	assert.Equal(rxtx.ErrorCode_CLIENT_ERROR, code)

	// Pull the message
	ackAndCheckResult := func(apnID int32, nasID int32, result rxtx.ErrorCode) {
		received := false
		var res *rxtx.DownstreamResponse
		var err error
		for !received {
			// Pull the message. The message should be returned
			res, err = r.GetMessage(context.Background(), &rxtx.DownstreamRequest{
				Origin: &rxtx.Origin{ApnId: apnID, NasId: []int32{nasID}},
				Type:   rxtx.MessageType_UDP,
			})
			assert.NoError(err)
			if res.Msg != nil {
				received = true
			}
			time.Sleep(100 * time.Millisecond)
		}
		assert.NotNil(res.Msg)
		_, err = r.Ack(context.Background(), &rxtx.AckRequest{
			MessageId: res.Msg.Id,
			Result:    result,
		})
		assert.NoError(err)
	}

	go ackAndCheckResult(1, 1, rxtx.ErrorCode_SUCCESS)
	// Send a message and wait for result
	msg.Type = rxtx.MessageType_UDP
	msg.Payload = []byte("hello there")
	code, err = r.Send(ctx, d, msg, true)
	assert.NoError(err)
	assert.Equal(rxtx.ErrorCode_SUCCESS, code)

	// This will be ignored
	go ackAndCheckResult(1, 1, rxtx.ErrorCode_NETWORK)
	// Send without waiting for result
	msg.Type = rxtx.MessageType_CoAPPull
	msg.Payload = []byte("hello there")
	msg.Coap = &rxtx.CoAPOptions{Path: ""}
	code, err = r.Send(ctx, d, msg, true)
	assert.NoError(err)
	assert.Equal(rxtx.ErrorCode_PENDING, code)

	go ackAndCheckResult(1, 1, rxtx.ErrorCode_SUCCESS)
	// Send without waiting.
	msg.Type = rxtx.MessageType_CoAPPush
	msg.Payload = []byte("hello there")
	msg.Coap = &rxtx.CoAPOptions{Path: ""}
	code, err = r.Send(ctx, d, msg, false)
	assert.NoError(err)
	assert.Equal(rxtx.ErrorCode_PENDING, code)

	timedOutCtx, timeDone := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer timeDone()
	time.Sleep(2 * time.Nanosecond)
	// Do a push that times out
	msg.Type = rxtx.MessageType_CoAPPush
	msg.Payload = []byte("hello there")
	msg.Coap = &rxtx.CoAPOptions{Path: ""}
	code, err = r.Send(timedOutCtx, d, msg, true)
	assert.Error(err)
	assert.Equal(rxtx.ErrorCode_TIMEOUT, code)
}

func setupCoap(assert *require.Assertions, t *testing.T) (*RxTxReceiver, model.Device) {
	publisher := make(chan model.DataMessage, 20)

	apnStore := sqlstore.NewMemoryAPNStore()
	assert.NoError(apnStore.CreateAPN(model.APN{ID: 1, Name: "test.apn"}))
	assert.NoError(apnStore.CreateNAS(model.NAS{ID: 1, CIDR: "10.0.0.0/16", Identifier: "NAS01", ApnID: 1}))

	apnConfig, err := storage.NewAPNCache(apnStore)
	assert.NoError(err)

	datastore := sqlstore.NewMemoryStore()
	te := storetest.NewTestEnvironment(t, datastore)
	d := model.NewDevice()
	d.ID = datastore.NewDeviceID()
	d.IMSI = 1001
	d.IMEI = 1001
	d.CollectionID = te.C1.ID
	d.Network.ApnID = 1
	d.Network.NasID = 1
	assert.NoError(datastore.CreateDevice(te.U1.ID, d))
	assert.NoError(apnStore.CreateAllocation(model.Allocation{
		IP:      net.ParseIP("10.0.0.1"),
		IMSI:    1001,
		IMEI:    1001,
		ApnID:   1,
		NasID:   1,
		Created: time.Now(),
	}))
	params := sqlstore.Parameters{
		ConnectionString: memoryDB,
		Type:             "sqlite3",
		CreateSchema:     true,
	}
	downstreamStore, err := sqlstore.NewDownstreamStore(datastore, params, 1, 1)
	assert.NoError(err)

	r := NewRxTxReceiver(apnConfig, datastore, apnStore, downstreamStore, publisher)
	assert.NotNil(r)

	return r, d
}
func TestExchange(t *testing.T) {
	defer purgeMessages()
	assert := require.New(t)
	r, d := setupCoap(assert, t)

	ctx, done := context.WithTimeout(context.Background(), 1*time.Second)
	defer done()

	inMsg := &rxtx.Message{
		Type:          rxtx.MessageType_CoAPPush,
		RemoteAddress: net.ParseIP("10.0.0.1"),
		RemotePort:    5863,
	}
	// Missing coap options
	msg, err := r.Exchange(ctx, &d, inMsg)
	assert.Error(err)
	assert.Nil(msg)

	inMsg.Coap = &rxtx.CoAPOptions{
		Code: int32(codes.GET),
		Path: "/whatever",
	}

	// Incorrect message type
	inMsg.Type = rxtx.MessageType_UDP
	msg, err = r.Exchange(ctx, &d, inMsg)
	assert.Error(err)
	assert.Nil(msg)

	inMsg.Type = rxtx.MessageType_CoAPPush

	wg := &sync.WaitGroup{}
	wg.Add(1)

	// Do an exchange in a new goroutine. The bottom (aka the listener calls
	// are performed in this thread.)
	go func() {
		ctx, done := context.WithTimeout(context.Background(), 1*time.Second)
		defer done()
		defer wg.Done()
		msg, err = r.Exchange(ctx, &d, inMsg)
		t.Log("Exchange is done")
		assert.NoError(err)
		assert.NotNil(msg)
		assert.Equal(msg.Payload, []byte("hello there"))
	}()

	// Wait for the message to appear in GetMessage
	received := false
	var res *rxtx.DownstreamResponse
	for !received {
		res, err = r.GetMessage(ctx, &rxtx.DownstreamRequest{
			Origin: &rxtx.Origin{ApnId: 1, NasId: []int32{1}},
			Type:   rxtx.MessageType_CoAPPush,
		})
		assert.NoError(err)
		received = (res.Msg != nil)
		if !received {
			time.Sleep(10 * time.Millisecond)
		}
	}
	t.Log("Ack and stuff")
	assert.Equal("/whatever", res.Msg.Coap.Path)
	r.Ack(ctx, &rxtx.AckRequest{MessageId: res.Msg.Id, Result: rxtx.ErrorCode_SUCCESS})
	t.Log("Put response")
	r.PutMessage(ctx, &rxtx.UpstreamRequest{
		Origin: &rxtx.Origin{ApnId: 1, NasId: []int32{1}},
		Msg: &rxtx.Message{
			Type: rxtx.MessageType_CoAPUpstream,
			Coap: &rxtx.CoAPOptions{
				Path:  "",
				Token: res.Msg.Coap.Token,
			},
			RemoteAddress: net.ParseIP("10.0.0.1"),
			RemotePort:    5683,
			Payload:       []byte("hello there"),
		},
	})
	t.Log("Wait for exchange to complete")
	wg.Wait()
}
