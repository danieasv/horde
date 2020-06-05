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
	"testing"

	"github.com/eesrc/horde/pkg/api/apipb"
	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/output"
	"github.com/eesrc/horde/pkg/storage/sqlstore"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestCollectionTags(t *testing.T) {
	assert := require.New(t)
	store := sqlstore.NewMemoryStore()
	fm := model.FieldMaskParameters{
		Default: "msisdn",
		Forced:  "msisdn",
	}
	collService := newCollectionService(store, fm, output.NewDummyManager(), newDummyDataStoreClient(), newDummyMessageSender())
	assert.NotNil(collService)

	user, _, ctx := createAuthenticatedContext(assert, store)

	c := model.NewCollection()
	c.ID = store.NewCollectionID()
	c.TeamID = user.PrivateTeamID

	assert.NoError(store.CreateCollection(user.ID, c))
	doTagTests(ctx, assert, c.ID.String(), true, c.ID.String(), tagFunctions{
		ListTags:   collService.ListCollectionTags,
		UpdateTag:  collService.UpdateCollectionTag,
		GetTag:     collService.GetCollectionTag,
		DeleteTag:  collService.DeleteCollectionTag,
		UpdateTags: collService.UpdateCollectionTags,
	})

}

func TestCreateRetrieveCollection(t *testing.T) {
	assert := require.New(t)
	store := sqlstore.NewMemoryStore()
	fm := model.FieldMaskParameters{
		Default: "msisdn",
		Forced:  "msisdn",
	}
	cs := newCollectionService(store, fm, output.NewDummyManager(), newDummyDataStoreClient(), newDummyMessageSender())
	assert.NotNil(cs)

	// Unauthenticated request should be rejected
	_, err := cs.CreateCollection(context.Background(), &apipb.Collection{})
	assert.Error(err)
	assert.Equal(codes.Unauthenticated.String(), status.Code(err).String())

	_, err = cs.RetrieveCollection(context.Background(), &apipb.RetrieveCollectionRequest{
		CollectionId: &wrappers.StringValue{Value: "0"},
	})
	assert.Error(err)
	assert.Equal(codes.Unauthenticated.String(), status.Code(err).String())

	user, _, ctx := createAuthenticatedContext(assert, store)

	// Nil request objects are rejected
	_, err = cs.CreateCollection(ctx, nil)
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	_, err = cs.RetrieveCollection(ctx, nil)
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	_, err = cs.RetrieveCollection(ctx, &apipb.RetrieveCollectionRequest{})
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Assign invalid formatted team ID. Should fail
	_, err = cs.CreateCollection(ctx, &apipb.Collection{
		TeamId: &wrappers.StringValue{Value: "invalid"},
	})
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Assign unknown team ID. Should be rejected
	_, err = cs.CreateCollection(ctx, &apipb.Collection{
		TeamId: &wrappers.StringValue{Value: "0"},
	})
	assert.Error(err)
	assert.Equal(codes.NotFound.String(), status.Code(err).String())

	// Assign an existing team that the user isn't a member of
	u2, _ := createUserAndToken(assert, model.AuthConnectID, store)
	u3, _ := createUserAndToken(assert, model.AuthConnectID, store)

	t1 := model.NewTeam()
	t1.ID = store.NewTeamID()
	t1.AddMember(model.NewMember(u2, model.AdminRole))
	t1.AddMember(model.NewMember(u3, model.MemberRole))
	assert.NoError(store.CreateTeam(t1))

	// This should return "not found" since the user isn't a member of the team
	_, err = cs.CreateCollection(ctx, &apipb.Collection{
		TeamId: &wrappers.StringValue{Value: t1.ID.String()},
	})
	assert.Error(err)
	assert.Equal(codes.NotFound.String(), status.Code(err).String())

	// Add user as regular member. Should get permission denied
	t1.AddMember(model.NewMember(user, model.MemberRole))
	assert.NoError(store.UpdateTeam(u2.ID, t1))
	_, err = cs.CreateCollection(ctx, &apipb.Collection{
		TeamId: &wrappers.StringValue{Value: t1.ID.String()},
	})
	assert.Error(err)
	assert.Equal(codes.PermissionDenied.String(), status.Code(err).String())

	// Make the user an admin. Should succeed.
	t1.UpdateMember(user.ID, model.AdminRole)
	assert.NoError(store.UpdateTeam(u2.ID, t1))
	res1, err := cs.CreateCollection(ctx, &apipb.Collection{
		TeamId: &wrappers.StringValue{Value: t1.ID.String()},
		Tags: map[string]string{
			"name": "Collection 1",
		},
	})
	assert.NoError(err)
	assert.NotNil(res1)
	assert.Equal(t1.ID.String(), res1.TeamId.Value)

	// Assign a firmware update but leave the team ID blank. The
	// private team for the user should be used.
	res2, err := cs.CreateCollection(ctx, &apipb.Collection{
		Firmware: &apipb.CollectionFirmware{
			Management: apipb.CollectionFirmware_collection,
		},
	})
	assert.NoError(err)
	assert.NotNil(res2)
	assert.NotNil(res2.TeamId)
	assert.Equal(user.PrivateTeamID.String(), res2.TeamId.Value)
	assert.NotNil(res2.Firmware)
	assert.Equal(apipb.CollectionFirmware_collection, res2.Firmware.Management)

	// Assign different management settings to the firmware for the collection
	res, err := cs.CreateCollection(ctx, &apipb.Collection{
		Firmware: &apipb.CollectionFirmware{
			Management: apipb.CollectionFirmware_device,
		},
	})
	assert.NoError(err)
	assert.Equal(apipb.CollectionFirmware_device, res.Firmware.Management)

	res, err = cs.CreateCollection(ctx, &apipb.Collection{
		Firmware: &apipb.CollectionFirmware{
			Management: apipb.CollectionFirmware_disabled,
		},
	})
	assert.NoError(err)
	assert.Equal(apipb.CollectionFirmware_disabled, res.Firmware.Management)

	// assign unknown type. Should get an error
	_, err = cs.CreateCollection(ctx, &apipb.Collection{
		Firmware: &apipb.CollectionFirmware{
			Management: apipb.CollectionFirmware_FirmwareManagement(100),
		},
	})
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Assign invalid tags. Should fail
	_, err = cs.CreateCollection(ctx, &apipb.Collection{
		Tags: map[string]string{
			"name": "The other",
			"":     "Invalid tag",
		},
	})
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Retrieve the collections created earlier. Should be the same as the
	// returned value
	c1, err := cs.RetrieveCollection(ctx, &apipb.RetrieveCollectionRequest{
		CollectionId: res1.CollectionId,
	})
	assert.NoError(err)
	assert.EqualValues(c1, res1)

	c2, err := cs.RetrieveCollection(ctx, &apipb.RetrieveCollectionRequest{
		CollectionId: res2.CollectionId,
	})
	assert.NoError(err)
	assert.EqualValues(c2, res2)

	// Retrieve unknown collection. Should fail
	_, err = cs.RetrieveCollection(ctx, &apipb.RetrieveCollectionRequest{
		CollectionId: &wrappers.StringValue{Value: "0"},
	})
	assert.Error(err)
	assert.Equal(codes.NotFound.String(), status.Code(err).String())
}

func TestDeleteCollection(t *testing.T) {
	assert := require.New(t)
	store := sqlstore.NewMemoryStore()
	fm := model.FieldMaskParameters{
		Default: "msisdn",
		Forced:  "msisdn",
	}
	cs := newCollectionService(store, fm, output.NewDummyManager(), newDummyDataStoreClient(), newDummyMessageSender())
	assert.NotNil(cs)

	// Unauthenticated request should be rejected
	_, err := cs.DeleteCollection(context.Background(), &apipb.DeleteCollectionRequest{
		CollectionId: &wrappers.StringValue{Value: "0"},
	})
	assert.Error(err)
	assert.Equal(codes.Unauthenticated.String(), status.Code(err).String())

	user, _, ctx := createAuthenticatedContext(assert, store)

	// Nil requests will be rejected
	_, err = cs.DeleteCollection(ctx, nil)
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())
	_, err = cs.DeleteCollection(ctx, &apipb.DeleteCollectionRequest{})
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Create a test collection in the store that we can delete
	tmpColl := model.NewCollection()
	tmpColl.ID = store.NewCollectionID()
	tmpColl.TeamID = user.PrivateTeamID
	tmpColl.SetTag("name", "Test 1")
	assert.NoError(store.CreateCollection(user.ID, tmpColl))

	res, err := cs.DeleteCollection(ctx, &apipb.DeleteCollectionRequest{
		CollectionId: &wrappers.StringValue{Value: tmpColl.ID.String()},
	})
	assert.NoError(err)
	assert.NotNil(res)
	assert.Equal(tmpColl.ID.String(), res.CollectionId.Value)
	assert.Equal(tmpColl.GetTag("name"), res.Tags["name"])

	// Deleting a 2nd time will return not found
	_, err = cs.DeleteCollection(ctx, &apipb.DeleteCollectionRequest{
		CollectionId: &wrappers.StringValue{Value: tmpColl.ID.String()},
	})
	assert.Error(err)
	assert.Equal(codes.NotFound.String(), status.Code(err).String())
}

func TestUpdateCollection(t *testing.T) {
	assert := require.New(t)
	store := sqlstore.NewMemoryStore()
	fm := model.FieldMaskParameters{
		Default: "msisdn",
		Forced:  "msisdn",
	}
	cs := newCollectionService(store, fm, output.NewDummyManager(), newDummyDataStoreClient(), newDummyMessageSender())
	assert.NotNil(cs)

	// Unauthenticated request should be rejected
	_, err := cs.UpdateCollection(context.Background(), &apipb.Collection{
		CollectionId: &wrappers.StringValue{Value: "0"},
	})
	assert.Error(err)
	assert.Equal(codes.Unauthenticated.String(), status.Code(err).String())

	user, _, ctx := createAuthenticatedContext(assert, store)

	// Nil requests will be rejected
	_, err = cs.UpdateCollection(ctx, nil)
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())
	_, err = cs.UpdateCollection(ctx, &apipb.Collection{})
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Unknown collection returns Not found
	_, err = cs.UpdateCollection(ctx, &apipb.Collection{
		CollectionId: &wrappers.StringValue{Value: "0"},
	})
	assert.Error(err)
	assert.Equal(codes.NotFound.String(), status.Code(err).String())

	// Create a test collection in the store
	tmpColl := model.NewCollection()
	tmpColl.ID = store.NewCollectionID()
	tmpColl.TeamID = user.PrivateTeamID
	tmpColl.SetTag("name", "Test 1")
	assert.NoError(store.CreateCollection(user.ID, tmpColl))

	// Invalid tag/value should fail
	_, err = cs.UpdateCollection(ctx, &apipb.Collection{
		CollectionId: &wrappers.StringValue{Value: tmpColl.ID.String()},
		Tags: map[string]string{
			"name": "should not change",
			"":     "invalid value",
		},
	})
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Update the collecton with a new tag
	res, err := cs.UpdateCollection(ctx, &apipb.Collection{
		CollectionId: &wrappers.StringValue{Value: tmpColl.ID.String()},
		Tags: map[string]string{
			"other": "value",
		},
	})
	assert.NoError(err)
	assert.Contains(res.Tags, "other")
	assert.Contains(res.Tags, "name")
	assert.Equal(res.Tags["name"], tmpColl.GetTag("name"))

	// Create a few firmware images that we can use
	fw1 := model.NewFirmware()
	fw1.ID = store.NewFirmwareID()
	fw1.CollectionID = tmpColl.ID
	fw1.Length = 100
	fw1.SHA256 = "aaaa"
	fw1.Filename = "file1.bin"
	fw1.Version = "1"
	fw1.SetTag("name", "firmware 1")
	assert.NoError(store.CreateFirmware(user.ID, fw1))

	fw2 := model.NewFirmware()
	fw2.ID = store.NewFirmwareID()
	fw2.CollectionID = tmpColl.ID
	fw2.Length = 200
	fw2.SHA256 = "bbbb"
	fw2.Filename = "file2.bin"
	fw2.SetTag("name", "firmware 2")
	fw2.Version = "2"
	assert.NoError(store.CreateFirmware(user.ID, fw2))

	// Update collection with firmware images, management and field mask
	res, err = cs.UpdateCollection(ctx, &apipb.Collection{
		CollectionId: &wrappers.StringValue{Value: tmpColl.ID.String()},
		Firmware: &apipb.CollectionFirmware{
			Management:        apipb.CollectionFirmware_collection,
			CurrentFirmwareId: &wrappers.StringValue{Value: fw1.ID.String()},
			TargetFirmwareId:  &wrappers.StringValue{Value: fw2.ID.String()},
		},
	})
	assert.NoError(err)
	assert.Equal(fw1.ID.String(), res.Firmware.CurrentFirmwareId.Value)
	assert.Equal(fw2.ID.String(), res.Firmware.TargetFirmwareId.Value)

	// Use unknown current firmware ID. Should fail.
	_, err = cs.UpdateCollection(ctx, &apipb.Collection{
		CollectionId: &wrappers.StringValue{Value: tmpColl.ID.String()},
		Firmware: &apipb.CollectionFirmware{
			CurrentFirmwareId: &wrappers.StringValue{Value: "1"},
		},
	})
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// ..ditto for target firmware
	_, err = cs.UpdateCollection(ctx, &apipb.Collection{
		CollectionId: &wrappers.StringValue{Value: tmpColl.ID.String()},
		Firmware: &apipb.CollectionFirmware{
			TargetFirmwareId: &wrappers.StringValue{Value: "1"},
		},
	})
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// ...and invalid current+target firmware ID
	_, err = cs.UpdateCollection(ctx, &apipb.Collection{
		CollectionId: &wrappers.StringValue{Value: tmpColl.ID.String()},
		Firmware: &apipb.CollectionFirmware{
			CurrentFirmwareId: &wrappers.StringValue{Value: "firmware-1.id"},
		},
	})
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	_, err = cs.UpdateCollection(ctx, &apipb.Collection{
		CollectionId: &wrappers.StringValue{Value: tmpColl.ID.String()},
		Firmware: &apipb.CollectionFirmware{
			TargetFirmwareId: &wrappers.StringValue{Value: "firmware-2.id"},
		},
	})
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Invalid management setting should also fail
	_, err = cs.UpdateCollection(ctx, &apipb.Collection{
		CollectionId: &wrappers.StringValue{Value: tmpColl.ID.String()},
		Firmware: &apipb.CollectionFirmware{
			Management: apipb.CollectionFirmware_FirmwareManagement(999),
		},
	})
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Create an output for the collection. This isn't used but changes in the
	// field mask requires interaction with existing collections.
	output := model.NewOutput()
	output.ID = store.NewOutputID()
	output.CollectionID = tmpColl.ID
	output.Type = "udp"
	output.Config = model.NewOutputConfig()
	output.Config["endpoint"] = "something"
	output.Enabled = true
	output.SetTag("name", "Dummy output")
	assert.NoError(store.CreateOutput(user.ID, output))

	// Set the field mask to something else
	res, err = cs.UpdateCollection(ctx, &apipb.Collection{
		CollectionId: &wrappers.StringValue{Value: tmpColl.ID.String()},
		FieldMask: &apipb.FieldMask{
			Imsi: &wrappers.BoolValue{Value: true},
		},
	})
	assert.NoError(err)
	assert.True(res.FieldMask.Imsi.Value)

	// Create a new team and assign that to the collection
	team := model.NewTeam()
	team.ID = store.NewTeamID()
	team.AddMember(model.NewMember(user, model.AdminRole))
	assert.NoError(store.CreateTeam(team))

	res, err = cs.UpdateCollection(ctx, &apipb.Collection{
		CollectionId: &wrappers.StringValue{Value: tmpColl.ID.String()},
		TeamId:       &wrappers.StringValue{Value: team.ID.String()},
	})
	assert.NoError(err)
	assert.Equal(team.ID.String(), res.TeamId.Value)

	// Assign a team that doesn't exist (ie not an admin)
	_, err = cs.UpdateCollection(ctx, &apipb.Collection{
		CollectionId: &wrappers.StringValue{Value: tmpColl.ID.String()},
		TeamId:       &wrappers.StringValue{Value: "xx"},
	})
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())
}

func TestListCollections(t *testing.T) {
	assert := require.New(t)
	store := sqlstore.NewMemoryStore()
	fm := model.FieldMaskParameters{
		Default: "msisdn",
		Forced:  "msisdn",
	}
	cs := newCollectionService(store, fm, output.NewDummyManager(), newDummyDataStoreClient(), newDummyMessageSender())
	assert.NotNil(cs)

	// Unauthenticated request should be rejected
	_, err := cs.ListCollections(context.Background(), &apipb.ListCollectionRequest{})
	assert.Error(err)
	assert.Equal(codes.Unauthenticated.String(), status.Code(err).String())

	user, _, ctx := createAuthenticatedContext(assert, store)

	// Nil requests will be rejected
	_, err = cs.ListCollections(ctx, nil)
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Create a test collection in the store
	tmpColl := model.NewCollection()
	tmpColl.ID = store.NewCollectionID()
	tmpColl.TeamID = user.PrivateTeamID
	tmpColl.SetTag("name", "Test 1")
	assert.NoError(store.CreateCollection(user.ID, tmpColl))

	res, err := cs.ListCollections(ctx, &apipb.ListCollectionRequest{})
	assert.NoError(err)
	assert.Len(res.Collections, 1)
}

func TestCollectionListMessages(t *testing.T) {
	assert := require.New(t)
	store := sqlstore.NewMemoryStore()

	fm := model.FieldMaskParameters{
		Default: "msisdn",
		Forced:  "msisdn",
	}
	cs := newCollectionService(store, fm, output.NewDummyManager(), newDummyDataStoreClient(), newDummyMessageSender())
	assert.NotNil(cs)

	user, _, ctx := createAuthenticatedContext(assert, store)

	// Create a test collection in the store
	tmpColl := model.NewCollection()
	tmpColl.ID = store.NewCollectionID()
	tmpColl.TeamID = user.PrivateTeamID
	tmpColl.SetTag("name", "Test 1")
	assert.NoError(store.CreateCollection(user.ID, tmpColl))

	// Nil request => error
	_, err := cs.ListCollectionMessages(ctx, nil)
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	_, err = cs.ListCollectionMessages(ctx, &apipb.ListMessagesRequest{})
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Unauthenticated => error
	_, err = cs.ListCollectionMessages(context.Background(), &apipb.ListMessagesRequest{
		CollectionId: &wrappers.StringValue{Value: tmpColl.ID.String()},
	})
	assert.Error(err)
	assert.Equal(codes.Unauthenticated.String(), status.Code(err).String())

	// Get messages. The default client will return 10 messages for us
	// so filtering have no effect
	res, err := cs.ListCollectionMessages(ctx, &apipb.ListMessagesRequest{
		CollectionId: &wrappers.StringValue{Value: tmpColl.ID.String()},
	})

	assert.NoError(err)
	assert.NotNil(res)
	assert.Len(res.Messages, 10)

	// Unknown collection ID => error
	_, err = cs.ListCollectionMessages(ctx, &apipb.ListMessagesRequest{
		CollectionId: &wrappers.StringValue{Value: "0"},
	})
	assert.Error(err)
	assert.Equal(codes.NotFound.String(), status.Code(err).String())

	// Invalid collection ID => error
	_, err = cs.ListCollectionMessages(ctx, &apipb.ListMessagesRequest{
		CollectionId: &wrappers.StringValue{Value: "xx_xx"},
	})
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())
}

func TestCollectionBroadcast(t *testing.T) {
	assert := require.New(t)
	store := sqlstore.NewMemoryStore()

	fm := model.FieldMaskParameters{
		Default: "msisdn",
		Forced:  "msisdn",
	}
	sender := newDummyMessageSender()

	cs := newCollectionService(store, fm, output.NewDummyManager(), newDummyDataStoreClient(), sender)
	assert.NotNil(cs)

	user, _, ctx := createAuthenticatedContext(assert, store)

	_, err := cs.BroadcastMessage(ctx, nil)
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	_, err = cs.BroadcastMessage(ctx, &apipb.SendMessageRequest{})
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	c := model.NewCollection()
	c.ID = store.NewCollectionID()
	c.TeamID = user.PrivateTeamID
	c.SetTag("name", "Test 1")
	assert.NoError(store.CreateCollection(user.ID, c))

	// even without any devices in the collection this counts as a valid
	// request. Exactly 0 messages should be sent to all the devices in the
	// collection.
	r := &apipb.SendMessageRequest{
		CollectionId: &wrappers.StringValue{Value: c.ID.String()},
		Port:         &wrappers.Int32Value{Value: 99},
		Payload:      []byte("Hello there"),
	}

	// Ensure requests are authenticated
	_, err = cs.BroadcastMessage(context.Background(), r)
	assert.Error(err)
	assert.Equal(codes.Unauthenticated.String(), status.Code(err).String())

	// Unknown collection ID => error
	r.CollectionId = &wrappers.StringValue{Value: "0123a"}
	_, err = cs.BroadcastMessage(ctx, r)
	assert.Error(err)

	r.CollectionId = &wrappers.StringValue{Value: c.ID.String()}

	// Invalid message parameters => error
	r.Transport = &wrappers.StringValue{Value: "carrier-pigeon"}
	_, err = cs.BroadcastMessage(ctx, r)
	assert.Error(err)

	r.Transport = &wrappers.StringValue{Value: "udp"}

	// Valid message should work
	res, err := cs.BroadcastMessage(ctx, r)
	assert.NoError(err)
	assert.NotNil(res)
	assert.NotNil(res.Errors)
	assert.Equal(int32(0), res.Sent)
	assert.Equal(int32(0), res.Failed)

	failDevice := 105
	failedKey := ""
	// Add a few devices to the collection.
	for i := 100; i < 110; i++ {
		d := model.NewDevice()
		d.IMSI = int64(i)
		d.IMEI = int64(i)
		d.ID = store.NewDeviceID()
		if failDevice == i {
			failedKey = d.ID.String()
		}
		d.CollectionID = c.ID
		assert.NoError(store.CreateDevice(user.ID, d))
	}
	sender.FailOnIMSI = int64(failDevice)

	res, err = cs.BroadcastMessage(ctx, r)
	assert.NoError(err)
	assert.NotNil(res)
	assert.NotNil(res.Errors)
	assert.Len(res.Errors, 1)
	// Should contain the ID of the failed device
	assert.Equal(failedKey, res.Errors[0].DeviceId.Value)
	assert.Equal(int32(9), res.Sent)
	assert.Equal(int32(1), res.Failed)
}
