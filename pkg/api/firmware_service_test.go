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
	"github.com/eesrc/horde/pkg/storage/fwimage"
	"github.com/eesrc/horde/pkg/storage/sqlstore"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Scaffolding for the firmware tests. These are all initialized
// the same way
type firmwareTestSetup struct {
	assert          *require.Assertions
	store           storage.DataStore
	firmwareService firmwareService
	user            model.User
	ctx             context.Context
	collection      model.Collection
	firmware        model.Firmware
	imageStore      storage.FirmwareImageStore
}

func newFirmwareTest(t *testing.T) firmwareTestSetup {
	ret := firmwareTestSetup{}

	ret.assert = require.New(t)
	var err error

	ret.store, err = sqlstore.NewSQLStore("sqlite3", "file::memory:?_foreign_keys=1&_cache=shared", true, 1, 1)
	ret.assert.NoError(err)

	ret.imageStore, err = fwimage.NewSQLStore(sqlstore.Parameters{
		Type:             "sqlite3",
		ConnectionString: "file::memory:?_foreign_keys=1&_cache=shared",
		CreateSchema:     true,
	})
	ret.assert.NoError(err)

	ret.firmwareService = newFirmwareService(ret.store, ret.imageStore)
	ret.assert.NotNil(ret.firmwareService)

	ret.user, _, ret.ctx = createAuthenticatedContext(ret.assert, ret.store)

	ret.collection = model.NewCollection()
	ret.collection.ID = ret.store.NewCollectionID()
	ret.collection.TeamID = ret.user.PrivateTeamID
	ret.assert.NoError(ret.store.CreateCollection(ret.user.ID, ret.collection))

	ret.firmware = model.NewFirmware()
	ret.firmware.ID = ret.store.NewFirmwareID()
	ret.firmware.CollectionID = ret.collection.ID
	ret.firmware.SHA256 = "FAFAFA"
	ret.firmware.Version = "1.0"
	ret.assert.NoError(ret.store.CreateFirmware(ret.user.ID, ret.firmware))

	return ret
}

func TestFirmwareTags(t *testing.T) {
	ft := newFirmwareTest(t)

	doTagTests(ft.ctx, ft.assert, ft.collection.ID.String(), true, ft.firmware.ID.String(), tagFunctions{
		ListTags:   ft.firmwareService.ListFirmwareTags,
		UpdateTag:  ft.firmwareService.UpdateFirmwareTag,
		GetTag:     ft.firmwareService.GetFirmwareTag,
		DeleteTag:  ft.firmwareService.DeleteFirmwareTag,
		UpdateTags: ft.firmwareService.UpdateFirmwareTags,
	})
}

// Factory for apipb.Firmware requests
type firmwareRequestFactory struct {
}

func (frf *firmwareRequestFactory) ValidRequest() interface{} {
	return &apipb.FirmwareRequest{}
}

func (frf *firmwareRequestFactory) SetCollection(req interface{}, cid *wrappers.StringValue) {
	req.(*apipb.FirmwareRequest).CollectionId = cid
}

func (frf *firmwareRequestFactory) SetIdentifier(req interface{}, oid *wrappers.StringValue) {
	req.(*apipb.FirmwareRequest).ImageId = oid
}

func TestRetrieveFirmware(t *testing.T) {
	ft := newFirmwareTest(t)

	genericRequestTests(tparam{
		AuthenticatedContext:      ft.ctx,
		Assert:                    ft.assert,
		CollectionID:              ft.collection.ID.String(),
		TestWithInvalidIdentifier: true,
		IdentifierID:              ft.firmware.ID.String(),
		RequestFactory:            &firmwareRequestFactory{},
		RequestFunc: func(ctx context.Context, req interface{}) (interface{}, error) {
			if req == nil {
				return ft.firmwareService.RetrieveFirmware(ctx, nil)
			}
			return ft.firmwareService.RetrieveFirmware(ctx, req.(*apipb.FirmwareRequest))
		},
	})
}

func TestFirmwareUsage(t *testing.T) {
	ft := newFirmwareTest(t)

	// Create two devices and set one as current and one as targeted
	d1 := model.NewDevice()
	d1.ID = ft.store.NewDeviceID()
	d1.IMSI = 4711
	d1.IMEI = 4711
	d1.Firmware.CurrentFirmwareID = ft.firmware.ID
	d1.Firmware.TargetFirmwareID = 0
	d1.CollectionID = ft.collection.ID
	ft.assert.NoError(ft.store.CreateDevice(ft.user.ID, d1))

	d2 := model.NewDevice()
	d2.ID = ft.store.NewDeviceID()
	d2.IMSI = 4712
	d2.IMEI = 4712
	d2.Firmware.CurrentFirmwareID = 0
	d2.Firmware.TargetFirmwareID = ft.firmware.ID
	d2.CollectionID = ft.collection.ID
	ft.assert.NoError(ft.store.CreateDevice(ft.user.ID, d2))

	genericRequestTests(
		tparam{
			AuthenticatedContext:      ft.ctx,
			Assert:                    ft.assert,
			CollectionID:              ft.collection.ID.String(),
			TestWithInvalidIdentifier: true,
			IdentifierID:              ft.firmware.ID.String(),
			RequestFactory:            &firmwareRequestFactory{},
			RequestFunc: func(ctx context.Context, req interface{}) (interface{}, error) {
				if req == nil {
					return ft.firmwareService.FirmwareUsage(ctx, nil)
				}
				return ft.firmwareService.FirmwareUsage(ctx, req.(*apipb.FirmwareRequest))
			}})

	req := &apipb.FirmwareRequest{
		CollectionId: &wrappers.StringValue{Value: ft.collection.ID.String()},
		ImageId:      &wrappers.StringValue{Value: ft.firmware.ID.String()},
	}
	res, err := ft.firmwareService.FirmwareUsage(ft.ctx, req)
	ft.assert.NoError(err)
	ft.assert.NotNil(res)
	ft.assert.Len(res.Current, 1)
	ft.assert.Len(res.Targeted, 1)
}

func TestDeleteFirmware(t *testing.T) {
	ft := newFirmwareTest(t)

	genericRequestTests(tparam{
		AuthenticatedContext:      ft.ctx,
		Assert:                    ft.assert,
		CollectionID:              ft.collection.ID.String(),
		TestWithInvalidIdentifier: true,
		IdentifierID:              ft.firmware.ID.String(),
		RequestFactory:            &firmwareRequestFactory{},
		RequestFunc: func(ctx context.Context, req interface{}) (interface{}, error) {
			if req == nil {
				return ft.firmwareService.DeleteFirmware(ctx, nil)
			}
			return ft.firmwareService.DeleteFirmware(ctx, req.(*apipb.FirmwareRequest))
		},
	})

	// ensure the firmware is removed at this time
	_, err := ft.store.RetrieveFirmware(ft.user.ID, ft.collection.ID, ft.firmware.ID)
	ft.assert.Equal(storage.ErrNotFound, err)

	// Removing it twice should return not found
	req := &apipb.FirmwareRequest{
		CollectionId: &wrappers.StringValue{Value: ft.collection.ID.String()},
		ImageId:      &wrappers.StringValue{Value: ft.firmware.ID.String()},
	}
	_, err = ft.firmwareService.DeleteFirmware(ft.ctx, req)
	ft.assert.Error(err)
	ft.assert.Equal(codes.NotFound.String(), status.Code(err).String())

	// Attempt to delete firmware in use should return FailedPrecondition (412)
	ft.assert.NoError(ft.store.CreateFirmware(ft.user.ID, ft.firmware))
	ft.collection.Firmware.Management = model.CollectionManagement
	ft.collection.Firmware.CurrentFirmwareID = ft.firmware.ID
	ft.assert.NoError(ft.store.UpdateCollection(ft.user.ID, ft.collection))

	d := model.NewDevice()
	d.ID = ft.store.NewDeviceID()
	d.CollectionID = ft.collection.ID
	d.Firmware.CurrentFirmwareID = ft.firmware.ID
	d.IMSI = 4711
	d.IMEI = 4711
	ft.assert.NoError(ft.store.CreateDevice(ft.user.ID, d))

	_, err = ft.firmwareService.DeleteFirmware(ft.ctx, req)
	ft.assert.Error(err)
	ft.assert.Equal(codes.FailedPrecondition.String(), status.Code(err).String())

	// Attempt to remove firmware when not administrator => error
	u2, _, _ := createAuthenticatedContext(ft.assert, ft.store)
	t2 := model.NewTeam()
	t2.AddMember(model.NewMember(ft.user, model.MemberRole))
	t2.AddMember(model.NewMember(u2, model.AdminRole))
	ft.assert.NoError(ft.store.CreateTeam(t2))

	c2 := model.NewCollection()
	c2.TeamID = t2.ID
	c2.ID = ft.store.NewCollectionID()
	ft.assert.NoError(ft.store.CreateCollection(u2.ID, c2))

	// Note that the SHA and version are the same as the other collection.
	// This is by design
	fw := model.NewFirmware()
	fw.ID = ft.store.NewFirmwareID()
	fw.CollectionID = c2.ID
	fw.SHA256 = "FAFAFA"
	fw.Version = "1.0"
	ft.assert.NoError(ft.store.CreateFirmware(u2.ID, fw))

	req.CollectionId = &wrappers.StringValue{Value: c2.ID.String()}
	req.ImageId = &wrappers.StringValue{Value: fw.ID.String()}
	_, err = ft.firmwareService.DeleteFirmware(ft.ctx, req)
	ft.assert.Error(err)
	ft.assert.Equal(codes.PermissionDenied.String(), status.Code(err).String())
}

// Factory for apipb.Firmware requests
type createFirmwareRequestFactory struct {
}

func (cfr *createFirmwareRequestFactory) ValidRequest() interface{} {
	return &apipb.CreateFirmwareRequest{
		Image:    []byte(fmt.Sprintf("hello there i'm an image time is %d", time.Now().UnixNano())),
		Filename: &wrappers.StringValue{Value: "file.bin"},
		Version:  &wrappers.StringValue{Value: fmt.Sprintf("%d", time.Now().UnixNano())},
		Tags: map[string]string{
			"Name": "The test image",
		},
	}
}

func (cfr *createFirmwareRequestFactory) SetCollection(req interface{}, cid *wrappers.StringValue) {
	req.(*apipb.CreateFirmwareRequest).CollectionId = cid
}

func (cfr *createFirmwareRequestFactory) SetIdentifier(req interface{}, oid *wrappers.StringValue) {
}

func TestCreateFirmware(t *testing.T) {
	ft := newFirmwareTest(t)

	genericRequestTests(tparam{
		AuthenticatedContext:      ft.ctx,
		Assert:                    ft.assert,
		CollectionID:              ft.collection.ID.String(),
		TestWithInvalidIdentifier: false,
		IdentifierID:              "",
		RequestFactory:            &createFirmwareRequestFactory{},
		RequestFunc: func(ctx context.Context, req interface{}) (interface{}, error) {
			if req == nil {
				return ft.firmwareService.CreateFirmware(ctx, nil)
			}
			return ft.firmwareService.CreateFirmware(ctx, req.(*apipb.CreateFirmwareRequest))
		}})

	// Missing image => error
	req := &apipb.CreateFirmwareRequest{
		CollectionId: &wrappers.StringValue{Value: ft.collection.ID.String()},
		Image:        nil,
		Filename:     &wrappers.StringValue{Value: "some_thing.bin"},
		Version:      &wrappers.StringValue{Value: "9.0"},
	}
	_, err := ft.firmwareService.CreateFirmware(ft.ctx, req)
	ft.assert.Error(err)
	ft.assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Missing filename => error
	req.Image = []byte("some thing here")
	req.Filename = &wrappers.StringValue{Value: "      "}
	_, err = ft.firmwareService.CreateFirmware(ft.ctx, req)
	ft.assert.Error(err)
	ft.assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Missing version => error
	req.Filename = &wrappers.StringValue{Value: "some_thing.bin"}
	req.Version = &wrappers.StringValue{Value: "      "}
	_, err = ft.firmwareService.CreateFirmware(ft.ctx, req)
	ft.assert.Error(err)
	ft.assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Invalid tags => error
	req.Version = &wrappers.StringValue{Value: "9.0"}
	req.Tags = map[string]string{
		"": "Invalid tag",
	}
	_, err = ft.firmwareService.CreateFirmware(ft.ctx, req)
	ft.assert.Error(err)
	ft.assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Duplicate image (version or sha) => error
	req.Tags = map[string]string{
		"Name": "Version nine",
	}
	res, err := ft.firmwareService.CreateFirmware(ft.ctx, req)
	ft.assert.NoError(err)
	ft.assert.NotNil(res)
	// Replace the contents (ie new sha) - should fail
	tmp := req.Image
	req.Image = []byte("a different file but same version")
	_, err = ft.firmwareService.CreateFirmware(ft.ctx, req)
	ft.assert.Error(err)
	ft.assert.Equal(codes.FailedPrecondition.String(), status.Code(err).String())

	// Same contents, different version - should fail
	req.Image = tmp
	req.Version = &wrappers.StringValue{Value: "10.0"}
	_, err = ft.firmwareService.CreateFirmware(ft.ctx, req)
	ft.assert.Error(err)
	ft.assert.Equal(codes.FailedPrecondition.String(), status.Code(err).String())

	// Not the owner => error
	u2, _, _ := createAuthenticatedContext(ft.assert, ft.store)
	t2 := model.NewTeam()
	t2.AddMember(model.NewMember(ft.user, model.MemberRole))
	t2.AddMember(model.NewMember(u2, model.AdminRole))
	ft.assert.NoError(ft.store.CreateTeam(t2))

	c2 := model.NewCollection()
	c2.TeamID = t2.ID
	c2.ID = ft.store.NewCollectionID()
	ft.assert.NoError(ft.store.CreateCollection(u2.ID, c2))

	req.CollectionId = &wrappers.StringValue{Value: c2.ID.String()}
	_, err = ft.firmwareService.CreateFirmware(ft.ctx, req)
	ft.assert.Error(err)
	ft.assert.Equal(codes.PermissionDenied.String(), status.Code(err).String())
}

type firmwareListRequestFactory struct {
}

func (flr *firmwareListRequestFactory) ValidRequest() interface{} {
	return &apipb.ListFirmwareRequest{}
}

func (flr *firmwareListRequestFactory) SetCollection(req interface{}, cid *wrappers.StringValue) {
	req.(*apipb.ListFirmwareRequest).CollectionId = cid
}

func (flr *firmwareListRequestFactory) SetIdentifier(req interface{}, oid *wrappers.StringValue) {
}

func TestListFirmware(t *testing.T) {
	ft := newFirmwareTest(t)

	genericRequestTests(tparam{
		AuthenticatedContext:      ft.ctx,
		Assert:                    ft.assert,
		CollectionID:              ft.collection.ID.String(),
		TestWithInvalidIdentifier: false,
		IdentifierID:              "",
		RequestFactory:            &firmwareListRequestFactory{},
		RequestFunc: func(ctx context.Context, req interface{}) (interface{}, error) {
			if req == nil {
				return ft.firmwareService.ListFirmware(ctx, nil)
			}
			return ft.firmwareService.ListFirmware(ctx, req.(*apipb.ListFirmwareRequest))
		}})
}

type firmwareFactory struct {
}

func (ff *firmwareFactory) ValidRequest() interface{} {
	return &apipb.Firmware{
		Version: &wrappers.StringValue{Value: fmt.Sprintf("%d", time.Now().UnixNano())},
		Tags: map[string]string{
			"Name": "The updated firmware",
		},
	}
}

func (ff *firmwareFactory) SetCollection(req interface{}, cid *wrappers.StringValue) {
	req.(*apipb.Firmware).CollectionId = cid
}

func (ff *firmwareFactory) SetIdentifier(req interface{}, oid *wrappers.StringValue) {
	req.(*apipb.Firmware).ImageId = oid
}
func TestUpdateFirmware(t *testing.T) {
	ft := newFirmwareTest(t)

	genericRequestTests(tparam{
		AuthenticatedContext:      ft.ctx,
		Assert:                    ft.assert,
		CollectionID:              ft.collection.ID.String(),
		TestWithInvalidIdentifier: true,
		IdentifierID:              ft.firmware.ID.String(),
		RequestFactory:            &firmwareFactory{},
		RequestFunc: func(ctx context.Context, req interface{}) (interface{}, error) {
			if req == nil {
				return ft.firmwareService.UpdateFirmware(ctx, nil)
			}
			return ft.firmwareService.UpdateFirmware(ctx, req.(*apipb.Firmware))
		}})

	req := &apipb.Firmware{
		ImageId:      &wrappers.StringValue{Value: ft.firmware.ID.String()},
		CollectionId: &wrappers.StringValue{Value: ft.collection.ID.String()},
		Version:      &wrappers.StringValue{Value: "99.9"},
	}
	// Modify filename, sha or length => error
	req.Filename = &wrappers.StringValue{Value: "readonly.bin"}
	_, err := ft.firmwareService.UpdateFirmware(ft.ctx, req)
	ft.assert.Error(err)
	ft.assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	req.Filename = nil
	req.Sha256 = &wrappers.StringValue{Value: "readonly"}
	_, err = ft.firmwareService.UpdateFirmware(ft.ctx, req)
	ft.assert.Error(err)
	ft.assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	req.Sha256 = nil
	req.Length = &wrappers.Int32Value{Value: 99}
	_, err = ft.firmwareService.UpdateFirmware(ft.ctx, req)
	ft.assert.Error(err)
	ft.assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())
	req.Length = nil

	// Invalid version => error
	req.Version = &wrappers.StringValue{Value: "   "}
	_, err = ft.firmwareService.UpdateFirmware(ft.ctx, req)
	ft.assert.Error(err)
	ft.assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Invalid tags => error
	req.Version = nil
	req.Tags = map[string]string{
		"": "Invalid name",
	}
	_, err = ft.firmwareService.UpdateFirmware(ft.ctx, req)
	ft.assert.Error(err)
	ft.assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Nothing to update => error
	req.Tags = nil
	_, err = ft.firmwareService.UpdateFirmware(ft.ctx, req)
	ft.assert.Error(err)
	ft.assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Update firmware when not admin => error
	u2, _, _ := createAuthenticatedContext(ft.assert, ft.store)
	t2 := model.NewTeam()
	t2.AddMember(model.NewMember(ft.user, model.MemberRole))
	t2.AddMember(model.NewMember(u2, model.AdminRole))
	ft.assert.NoError(ft.store.CreateTeam(t2))

	c2 := model.NewCollection()
	c2.TeamID = t2.ID
	c2.ID = ft.store.NewCollectionID()
	ft.assert.NoError(ft.store.CreateCollection(u2.ID, c2))

	fw := model.NewFirmware()
	fw.ID = ft.store.NewFirmwareID()
	fw.CollectionID = c2.ID
	fw.SHA256 = "FAFAFA"
	fw.Version = "1.0"
	ft.assert.NoError(ft.store.CreateFirmware(u2.ID, fw))

	req.CollectionId = &wrappers.StringValue{Value: c2.ID.String()}
	req.ImageId = &wrappers.StringValue{Value: fw.ID.String()}
	req.Version = &wrappers.StringValue{Value: "888.0"}
	_, err = ft.firmwareService.UpdateFirmware(ft.ctx, req)
	ft.assert.Error(err)
	ft.assert.Equal(codes.PermissionDenied.String(), status.Code(err).String())
}
