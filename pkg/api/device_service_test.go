package api

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
	"fmt"
	"testing"
	"time"

	"github.com/eesrc/horde/pkg/api/apipb"
	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
	"github.com/eesrc/horde/pkg/storage/sqlstore"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type deviceTestSetup struct {
	assert        *require.Assertions
	store         storage.DataStore
	deviceService deviceService
	user          model.User
	ctx           context.Context
	collection    model.Collection
	device        model.Device
	sender        *dummySender
}

func newDeviceTest(t *testing.T) deviceTestSetup {
	ret := deviceTestSetup{}
	ret.assert = require.New(t)

	ret.store = sqlstore.NewMemoryStore()
	ret.sender = newDummyMessageSender()
	ret.deviceService = newDeviceService(ret.store, newDummyDataStoreClient(), ret.sender)
	ret.assert.NotNil(ret.deviceService)

	ret.user, _, ret.ctx = createAuthenticatedContext(ret.assert, ret.store)

	ret.collection = model.NewCollection()
	ret.collection.ID = ret.store.NewCollectionID()
	ret.collection.TeamID = ret.user.PrivateTeamID
	ret.assert.NoError(ret.store.CreateCollection(ret.user.ID, ret.collection))

	ret.device = model.NewDevice()
	ret.device.IMSI = 4711
	ret.device.IMEI = 4711
	ret.device.ID = ret.store.NewDeviceID()
	ret.device.CollectionID = ret.collection.ID
	ret.assert.NoError(ret.store.CreateDevice(ret.user.ID, ret.device))
	return ret
}

func TestDeviceTags(t *testing.T) {
	dt := newDeviceTest(t)
	doTagTests(dt.ctx, dt.assert, dt.collection.ID.String(), true, dt.device.ID.String(), tagFunctions{
		ListTags:   dt.deviceService.ListDeviceTags,
		UpdateTag:  dt.deviceService.UpdateDeviceTag,
		GetTag:     dt.deviceService.GetDeviceTag,
		DeleteTag:  dt.deviceService.DeleteDeviceTag,
		UpdateTags: dt.deviceService.UpdateDeviceTags,
	})
}

type listMessagesResponseFactory struct {
}

func (f *listMessagesResponseFactory) ValidRequest() interface{} {
	return &apipb.ListMessagesRequest{}
}

func (f *listMessagesResponseFactory) SetCollection(req interface{}, cid *wrappers.StringValue) {
	req.(*apipb.ListMessagesRequest).CollectionId = cid
}

func (f *listMessagesResponseFactory) SetIdentifier(req interface{}, oid *wrappers.StringValue) {
	req.(*apipb.ListMessagesRequest).DeviceId = oid
}
func TestRetrieveMessageList(t *testing.T) {
	dt := newDeviceTest(t)

	genericRequestTests(tparam{
		AuthenticatedContext:      dt.ctx,
		Assert:                    dt.assert,
		CollectionID:              dt.collection.ID.String(),
		IdentifierID:              dt.device.ID.String(),
		TestWithInvalidIdentifier: true,
		RequestFactory:            &listMessagesResponseFactory{},
		RequestFunc: func(ctx context.Context, req interface{}) (interface{}, error) {
			if req == nil {
				return dt.deviceService.ListDeviceMessages(ctx, nil)
			}
			return dt.deviceService.ListDeviceMessages(ctx, req.(*apipb.ListMessagesRequest))
		}})
}

type sendMessageRequestFactory struct {
}

func (f *sendMessageRequestFactory) ValidRequest() interface{} {
	return &apipb.SendMessageRequest{
		Port:    &wrappers.Int32Value{Value: 8080},
		Payload: []byte("hello there"),
	}
}

func (f *sendMessageRequestFactory) SetCollection(req interface{}, cid *wrappers.StringValue) {
	req.(*apipb.SendMessageRequest).CollectionId = cid
}

func (f *sendMessageRequestFactory) SetIdentifier(req interface{}, oid *wrappers.StringValue) {
	req.(*apipb.SendMessageRequest).DeviceId = oid
}
func TestDeviceSendMessage(t *testing.T) {
	dt := newDeviceTest(t)

	genericRequestTests(tparam{
		AuthenticatedContext:      dt.ctx,
		Assert:                    dt.assert,
		CollectionID:              dt.collection.ID.String(),
		IdentifierID:              dt.device.ID.String(),
		TestWithInvalidIdentifier: true,
		RequestFactory:            &sendMessageRequestFactory{},
		RequestFunc: func(ctx context.Context, req interface{}) (interface{}, error) {
			if req == nil {
				return dt.deviceService.SendMessage(ctx, nil)
			}
			return dt.deviceService.SendMessage(ctx, req.(*apipb.SendMessageRequest))
		}})

	// Invalid message (missing port) => error
	r := &apipb.SendMessageRequest{
		CollectionId: &wrappers.StringValue{Value: dt.collection.ID.String()},
		DeviceId:     &wrappers.StringValue{Value: dt.device.ID.String()},
		Payload:      []byte("hello there"),
	}
	_, err := dt.deviceService.SendMessage(dt.ctx, r)
	dt.assert.Error(err)
	dt.assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	r.Port = &wrappers.Int32Value{Value: 8080}
	// Provoke error sending
	dt.sender.FailOnIMSI = dt.device.IMSI
	_, err = dt.deviceService.SendMessage(dt.ctx, r)
	dt.assert.Error(err)
	dt.assert.Equal(codes.AlreadyExists.String(), status.Code(err).String())
	dt.sender.FailOnIMSI = -1

}

type listDevicesRequestFactory struct {
}

func (f *listDevicesRequestFactory) ValidRequest() interface{} {
	return &apipb.ListDevicesRequest{}
}

func (f *listDevicesRequestFactory) SetCollection(req interface{}, cid *wrappers.StringValue) {
	req.(*apipb.ListDevicesRequest).CollectionId = cid
}

func (f *listDevicesRequestFactory) SetIdentifier(req interface{}, oid *wrappers.StringValue) {
}

func TestListDevices(t *testing.T) {
	dt := newDeviceTest(t)

	genericRequestTests(tparam{
		AuthenticatedContext:      dt.ctx,
		Assert:                    dt.assert,
		CollectionID:              dt.collection.ID.String(),
		IdentifierID:              "",
		TestWithInvalidIdentifier: false,
		RequestFactory:            &listDevicesRequestFactory{},
		RequestFunc: func(ctx context.Context, req interface{}) (interface{}, error) {
			if req == nil {
				return dt.deviceService.ListDevices(ctx, nil)
			}
			return dt.deviceService.ListDevices(ctx, req.(*apipb.ListDevicesRequest))
		}})

	r := &apipb.ListDevicesRequest{
		CollectionId: &wrappers.StringValue{Value: dt.collection.ID.String()},
	}
	res, err := dt.deviceService.ListDevices(dt.ctx, r)
	dt.assert.NoError(err)
	dt.assert.NotNil(res)
	dt.assert.Len(res.Devices, 1)
	dt.assert.Equal(fmt.Sprintf("%d", dt.device.IMSI), res.Devices[0].Imsi.Value)

}

type deviceRequestFactory struct {
}

func (f *deviceRequestFactory) ValidRequest() interface{} {
	return &apipb.DeviceRequest{}
}

func (f *deviceRequestFactory) SetCollection(req interface{}, cid *wrappers.StringValue) {
	req.(*apipb.DeviceRequest).CollectionId = cid
}

func (f *deviceRequestFactory) SetIdentifier(req interface{}, oid *wrappers.StringValue) {
	req.(*apipb.DeviceRequest).DeviceId = oid
}

func TestRetrieveDevice(t *testing.T) {
	dt := newDeviceTest(t)
	genericRequestTests(tparam{
		AuthenticatedContext:      dt.ctx,
		Assert:                    dt.assert,
		CollectionID:              dt.collection.ID.String(),
		IdentifierID:              dt.device.ID.String(),
		TestWithInvalidIdentifier: true,
		RequestFactory:            &deviceRequestFactory{},
		RequestFunc: func(ctx context.Context, req interface{}) (interface{}, error) {
			if req == nil {
				return dt.deviceService.RetrieveDevice(ctx, nil)
			}
			return dt.deviceService.RetrieveDevice(ctx, req.(*apipb.DeviceRequest))
		}})
}

func TestDeleteDevice(t *testing.T) {
	dt := newDeviceTest(t)
	genericRequestTests(tparam{
		AuthenticatedContext:      dt.ctx,
		Assert:                    dt.assert,
		CollectionID:              dt.collection.ID.String(),
		IdentifierID:              dt.device.ID.String(),
		TestWithInvalidIdentifier: true,
		RequestFactory:            &deviceRequestFactory{},
		RequestFunc: func(ctx context.Context, req interface{}) (interface{}, error) {
			if req == nil {
				return dt.deviceService.DeleteDevice(ctx, nil)
			}
			return dt.deviceService.DeleteDevice(ctx, req.(*apipb.DeviceRequest))
		}})

	// Should be removed from store
	_, err := dt.store.RetrieveDevice(dt.user.ID, dt.collection.ID, dt.device.ID)
	dt.assert.Equal(storage.ErrNotFound, err)
}

type deviceFactory struct {
}

func (f *deviceFactory) ValidRequest() interface{} {
	return &apipb.Device{
		Imsi: &wrappers.StringValue{Value: fmt.Sprintf("%d", time.Now().UnixNano())},
		Imei: &wrappers.StringValue{Value: fmt.Sprintf("%d", time.Now().UnixNano())},
	}
}

func (f *deviceFactory) SetCollection(req interface{}, cid *wrappers.StringValue) {
	req.(*apipb.Device).CollectionId = cid
}

func (f *deviceFactory) SetIdentifier(req interface{}, oid *wrappers.StringValue) {
	// Nothing - this is assigned by service
}

func TestCreateDevice(t *testing.T) {
	dt := newDeviceTest(t)
	genericRequestTests(tparam{
		AuthenticatedContext:      dt.ctx,
		Assert:                    dt.assert,
		CollectionID:              dt.collection.ID.String(),
		IdentifierID:              "",
		TestWithInvalidIdentifier: false,
		RequestFactory:            &deviceFactory{},
		RequestFunc: func(ctx context.Context, req interface{}) (interface{}, error) {
			if req == nil {
				return dt.deviceService.CreateDevice(ctx, nil)
			}
			return dt.deviceService.CreateDevice(ctx, req.(*apipb.Device))
		}})

	r := &apipb.Device{}
	// Missing IMSI and/or IMEI => error
	r.CollectionId = &wrappers.StringValue{Value: dt.collection.ID.String()}
	r.Imsi = nil
	r.Imei = nil
	_, err := dt.deviceService.CreateDevice(dt.ctx, r)
	dt.assert.Error(err)
	dt.assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	r.Imsi = &wrappers.StringValue{Value: "4712"}
	r.Imei = nil
	_, err = dt.deviceService.CreateDevice(dt.ctx, r)
	dt.assert.Error(err)
	dt.assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Invalid IMSI and IMEI values => error
	r.Imsi = &wrappers.StringValue{Value: "-4712"}
	r.Imei = &wrappers.StringValue{Value: "4712"}
	_, err = dt.deviceService.CreateDevice(dt.ctx, r)
	dt.assert.Error(err)
	dt.assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	r.Imsi = &wrappers.StringValue{Value: "4712"}
	r.Imei = &wrappers.StringValue{Value: "-4712"}
	_, err = dt.deviceService.CreateDevice(dt.ctx, r)
	dt.assert.Error(err)
	dt.assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	r.Imsi = nil
	r.Imei = &wrappers.StringValue{Value: "4712"}
	_, err = dt.deviceService.CreateDevice(dt.ctx, r)
	dt.assert.Error(err)
	dt.assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	r.Imsi = &wrappers.StringValue{Value: "4712"}
	// Invalid tag value => error
	r.Tags = map[string]string{
		"name":  "some name",
		"value": "some value",
		"":      "invalid name",
	}
	_, err = dt.deviceService.CreateDevice(dt.ctx, r)
	dt.assert.Error(err)
	dt.assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Proper tags - should succeed
	r.Tags = map[string]string{
		"name": "test device",
	}

	res, err := dt.deviceService.CreateDevice(dt.ctx, r)
	dt.assert.NoError(err)
	dt.assert.NotNil(res)
	dt.assert.Equal(res.CollectionId.Value, dt.collection.ID.String())

	// Create a 2nd collection. Not an admin => error
	// 2nd collection => duplicate IMSI/IMEI => error
	u2, _, ctx2 := createAuthenticatedContext(dt.assert, dt.store)
	t1 := model.NewTeam()
	t1.AddMember(model.NewMember(dt.user, model.MemberRole))
	t1.AddMember(model.NewMember(u2, model.AdminRole))
	dt.assert.NoError(dt.store.CreateTeam(t1))

	c2 := model.NewCollection()
	c2.ID = dt.store.NewCollectionID()
	c2.TeamID = t1.ID
	c2.SetTag("name", "Test 2")

	dt.assert.NoError(dt.store.CreateCollection(u2.ID, c2))

	r = &apipb.Device{
		CollectionId: &wrappers.StringValue{Value: c2.ID.String()},
		Imsi:         &wrappers.StringValue{Value: "4712"},
		Imei:         &wrappers.StringValue{Value: "4712"},
	}
	_, err = dt.deviceService.CreateDevice(dt.ctx, r)
	dt.assert.Error(err)
	dt.assert.Equal(codes.PermissionDenied.String(), status.Code(err).String())

	r.Imsi.Value = "4711"
	r.Imei.Value = "4712"
	_, err = dt.deviceService.CreateDevice(ctx2, r)
	dt.assert.Error(err)
	dt.assert.Equal(codes.AlreadyExists.String(), status.Code(err).String())

	r.Imsi.Value = "4712"
	r.Imei.Value = "4711"
	_, err = dt.deviceService.CreateDevice(ctx2, r)
	dt.assert.Error(err)
	dt.assert.Equal(codes.AlreadyExists.String(), status.Code(err).String())

	r.Imsi.Value = "4713"
	r.Imei.Value = "4713"
	_, err = dt.deviceService.CreateDevice(ctx2, r)
	dt.assert.NoError(err)
}

func TestDeviceUpdate(t *testing.T) {
	assert := require.New(t)
	store := sqlstore.NewMemoryStore()
	sender := newDummyMessageSender()
	deviceService := newDeviceService(store, newDummyDataStoreClient(), sender)
	assert.NotNil(deviceService)

	user, _, ctx := createAuthenticatedContext(assert, store)

	// Create a test collection and device in the store
	c := model.NewCollection()
	c.ID = store.NewCollectionID()
	c.TeamID = user.PrivateTeamID
	c.Firmware.Management = model.CollectionManagement
	c.SetTag("name", "Test 1")
	assert.NoError(store.CreateCollection(user.ID, c))
	// ..and a test device
	d := model.NewDevice()
	d.ID = store.NewDeviceID()
	d.CollectionID = c.ID
	d.IMSI = 1
	d.IMEI = 2
	assert.NoError(store.CreateDevice(user.ID, d))

	// Completely empty request object should do no update
	r := &apipb.UpdateDeviceRequest{}
	r.ExistingCollectionId = &wrappers.StringValue{Value: c.ID.String()}
	r.DeviceId = &wrappers.StringValue{Value: d.ID.String()}
	_, err := deviceService.UpdateDevice(ctx, r)
	assert.NoError(err)

	// Unauthenticated request
	_, err = deviceService.UpdateDevice(context.Background(), r)
	assert.Error(err)
	assert.Equal(codes.Unauthenticated.String(), status.Code(err).String())

	// No device ID or no collection ID => error
	r.ExistingCollectionId = nil
	_, err = deviceService.UpdateDevice(ctx, r)
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	r.DeviceId = nil
	r.ExistingCollectionId = &wrappers.StringValue{Value: c.ID.String()}
	_, err = deviceService.UpdateDevice(ctx, r)
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	r.DeviceId = &wrappers.StringValue{Value: d.ID.String()}

	// Unknown device => error
	r.DeviceId = &wrappers.StringValue{Value: "0"}
	_, err = deviceService.UpdateDevice(ctx, r)
	assert.Error(err)
	assert.Equal(codes.NotFound.String(), status.Code(err).String())

	// Uknown collection => error
	r.DeviceId = &wrappers.StringValue{Value: d.ID.String()}
	r.CollectionId = &wrappers.StringValue{Value: "0"}
	_, err = deviceService.UpdateDevice(ctx, r)
	assert.Error(err)
	assert.Equal(codes.NotFound.String(), status.Code(err).String())
	r.CollectionId = nil
	// Updating firmware when it is collection-managed => error
	c.Firmware.Management = model.CollectionManagement
	assert.NoError(store.UpdateCollection(user.ID, c))

	r.ExistingCollectionId = &wrappers.StringValue{Value: c.ID.String()}
	r.Firmware = &apipb.FirmwareMetadata{
		TargetFirmwareId: &wrappers.StringValue{Value: "0"},
	}
	_, err = deviceService.UpdateDevice(ctx, r)
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Invalid IMSI, IMEI => error
	r.Firmware = nil
	r.Imsi = &wrappers.StringValue{Value: fmt.Sprintf("%d", d.IMSI)}
	r.Imei = &wrappers.StringValue{Value: fmt.Sprintf("%d", -d.IMEI)}
	_, err = deviceService.UpdateDevice(ctx, r)
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	r.Imsi = &wrappers.StringValue{Value: fmt.Sprintf("%d", -d.IMSI)}
	r.Imei = &wrappers.StringValue{Value: fmt.Sprintf("%d", d.IMEI)}
	_, err = deviceService.UpdateDevice(ctx, r)
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Valid IMSI, IMEI => ok
	r.Imsi = &wrappers.StringValue{Value: "4712"}
	r.Imei = &wrappers.StringValue{Value: "4712"}
	t.Logf("Updating %+v", r)

	_, err = deviceService.UpdateDevice(ctx, r)
	assert.NoError(err)

	// Valid tags => ok
	r.Tags = map[string]string{
		"name":       "value",
		"other name": "other value",
	}
	_, err = deviceService.UpdateDevice(ctx, r)
	assert.NoError(err)

	// Invalid tags => error
	r.Tags = map[string]string{
		"name":       "value",
		"other name": "other value",
		"":           "heh",
	}
	_, err = deviceService.UpdateDevice(ctx, r)
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Update collection to a new one
	t1 := model.NewTeam()
	t1.AddMember(model.NewMember(user, model.AdminRole))
	assert.NoError(store.CreateTeam(t1))

	c2 := model.NewCollection()
	c2.ID = store.NewCollectionID()
	c2.TeamID = t1.ID
	c2.SetTag("name", "Test 2")
	assert.NoError(store.CreateCollection(user.ID, c2))

	r.Tags = nil
	r.Imsi = nil
	r.Imei = nil
	r.CollectionId = &wrappers.StringValue{Value: c2.ID.String()}
	_, err = deviceService.UpdateDevice(ctx, r)
	assert.NoError(err)
	//Update the existing ID since it has changed
	r.ExistingCollectionId = &wrappers.StringValue{Value: c2.ID.String()}

	// Attempt assignment to a collection where the user is not the owner
	u2, _ := createUserAndToken(assert, model.AuthConnectID, store)
	c3 := model.NewCollection()
	c3.ID = store.NewCollectionID()
	c3.TeamID = u2.PrivateTeamID
	c3.SetTag("name", "Test 3")
	assert.NoError(store.CreateCollection(u2.ID, c3))

	r.CollectionId = &wrappers.StringValue{Value: c3.ID.String()}
	_, err = deviceService.UpdateDevice(ctx, r)
	assert.Error(err)
	assert.Equal(codes.NotFound.String(), status.Code(err).String())

	// Attempt assignment to an invalid collection ID
	r.CollectionId = &wrappers.StringValue{Value: "invalid_key"}
	_, err = deviceService.UpdateDevice(ctx, r)
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Collection 2 isn't managed so it should be possible to set the firmware for it.
	fw1 := model.NewFirmware()
	fw1.ID = store.NewFirmwareID()
	fw1.CollectionID = c2.ID
	fw1.Length = 100
	fw1.SHA256 = "aaaa"
	fw1.Filename = "file1.bin"
	fw1.Version = "1"
	fw1.SetTag("name", "firmware 1")
	assert.NoError(store.CreateFirmware(user.ID, fw1))

	fw2 := model.NewFirmware()
	fw2.ID = store.NewFirmwareID()
	fw2.CollectionID = c2.ID
	fw2.Length = 200
	fw2.SHA256 = "bbbb"
	fw2.Filename = "file2.bin"
	fw2.SetTag("name", "firmware 2")
	fw2.Version = "2"
	assert.NoError(store.CreateFirmware(user.ID, fw2))

	r.CollectionId = nil
	// Assign fw1 as current, fw2 as target. Result should be "pending"
	// assign fw2 as current and fw2 as target. Resource should be "current
	r.Firmware = &apipb.FirmwareMetadata{
		CurrentFirmwareId: &wrappers.StringValue{Value: fw1.ID.String()},
		TargetFirmwareId:  &wrappers.StringValue{Value: fw2.ID.String()},
	}
	res, err := deviceService.UpdateDevice(ctx, r)
	assert.NoError(err)
	assert.Equal("Pending", res.Firmware.State.Value)

	r.Firmware = &apipb.FirmwareMetadata{
		CurrentFirmwareId: &wrappers.StringValue{Value: fw2.ID.String()},
		TargetFirmwareId:  &wrappers.StringValue{Value: fw2.ID.String()},
	}
	res, err = deviceService.UpdateDevice(ctx, r)
	assert.NoError(err)
	assert.Equal("Current", res.Firmware.State.Value)

	// Assign an unknown firmware ID as target. Should get error
	r.Firmware = &apipb.FirmwareMetadata{
		TargetFirmwareId: &wrappers.StringValue{Value: "1"},
	}
	_, err = deviceService.UpdateDevice(ctx, r)
	assert.Error(err)
	assert.Equal(codes.NotFound.String(), status.Code(err).String())

	r.Firmware = &apipb.FirmwareMetadata{
		CurrentFirmwareId: &wrappers.StringValue{Value: "1"},
	}
	_, err = deviceService.UpdateDevice(ctx, r)
	assert.Error(err)
	assert.Equal(codes.NotFound.String(), status.Code(err).String())

	// Then invalid values. Should get errors
	r.Firmware = &apipb.FirmwareMetadata{
		TargetFirmwareId: &wrappers.StringValue{Value: "-1"},
	}
	_, err = deviceService.UpdateDevice(ctx, r)
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	r.Firmware = &apipb.FirmwareMetadata{
		CurrentFirmwareId: &wrappers.StringValue{Value: "-1"},
	}
	_, err = deviceService.UpdateDevice(ctx, r)
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())
}

func TestClearFirmwareError(t *testing.T) {
	assert := require.New(t)
	store := sqlstore.NewMemoryStore()
	sender := newDummyMessageSender()
	deviceService := newDeviceService(store, newDummyDataStoreClient(), sender)
	assert.NotNil(deviceService)

	user, _, ctx := createAuthenticatedContext(assert, store)

	// Create a test collection and device in the store
	c := model.NewCollection()
	c.ID = store.NewCollectionID()
	c.TeamID = user.PrivateTeamID
	c.Firmware.Management = model.CollectionManagement
	c.SetTag("name", "Test 1")
	assert.NoError(store.CreateCollection(user.ID, c))
	// ..and a test device
	d := model.NewDevice()
	d.ID = store.NewDeviceID()
	d.CollectionID = c.ID
	d.Firmware.State = model.TimedOut
	d.IMSI = 1
	d.IMEI = 2
	assert.NoError(store.CreateDevice(user.ID, d))

	r := &apipb.DeviceRequest{
		CollectionId: &wrappers.StringValue{Value: c.ID.String()},
		DeviceId:     &wrappers.StringValue{Value: d.ID.String()},
	}
	_, err := deviceService.ClearFirmwareError(ctx, r)
	assert.NoError(err)

	// State should be pending in store
	tmp, err := store.RetrieveDevice(user.ID, c.ID, d.ID)
	assert.NoError(err)
	assert.Equal(model.Pending, tmp.Firmware.State)

	// Request with missing collection and device id => error
	_, err = deviceService.ClearFirmwareError(ctx, nil)
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	_, err = deviceService.ClearFirmwareError(ctx, &apipb.DeviceRequest{})
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	_, err = deviceService.ClearFirmwareError(ctx, &apipb.DeviceRequest{CollectionId: r.CollectionId})
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Unauthenticated => error
	_, err = deviceService.ClearFirmwareError(context.Background(), r)
	assert.Equal(codes.Unauthenticated.String(), status.Code(err).String())

	// Unknown device or collection  => error
	_, err = deviceService.ClearFirmwareError(ctx, &apipb.DeviceRequest{
		CollectionId: r.CollectionId,
		DeviceId:     &wrappers.StringValue{Value: "1"},
	})
	assert.Error(err)
	assert.Equal(codes.NotFound.String(), status.Code(err).String())

	_, err = deviceService.ClearFirmwareError(ctx, &apipb.DeviceRequest{
		CollectionId: &wrappers.StringValue{Value: "1"},
		DeviceId:     r.DeviceId,
	})
	assert.Error(err)
	assert.Equal(codes.NotFound.String(), status.Code(err).String())
}
