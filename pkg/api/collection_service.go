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

	"github.com/ExploratoryEngineering/logging"
	"github.com/eesrc/horde/pkg/addons/magpie/datastore"
	"github.com/eesrc/horde/pkg/api/apipb"
	"github.com/eesrc/horde/pkg/api/apitoolbox"
	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/output"
	"github.com/eesrc/horde/pkg/storage"
	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// newCollectionService creates a new service for the collection resource(s)
func newCollectionService(
	store storage.DataStore,
	fieldMask model.FieldMaskParameters,
	outputManager output.Manager,
	dataStoreClient datastore.DataStoreClient,
	messageSender DownstreamMessageSender) collectionService {
	return collectionService{
		store:           store,
		fieldMask:       fieldMask,
		outputManager:   outputManager,
		dataStoreClient: dataStoreClient,
		messageSender:   messageSender,
		defaultGrpcAuth: defaultGrpcAuth{Store: store},
	}
}

type collectionService struct {
	store           storage.DataStore
	fieldMask       model.FieldMaskParameters
	outputManager   output.Manager
	dataStoreClient datastore.DataStoreClient
	messageSender   DownstreamMessageSender

	defaultGrpcAuth
}

func (s *collectionService) getFirmwareManagement(setting *apipb.CollectionFirmware) (model.FirmwareManagementSetting, error) {
	switch setting.Management {
	case apipb.CollectionFirmware_collection:
		return model.CollectionManagement, nil
	case apipb.CollectionFirmware_device:
		return model.DeviceManagement, nil
	case apipb.CollectionFirmware_disabled:
		return model.DisabledManagement, nil
	default:
		return model.DisabledManagement, status.Error(codes.InvalidArgument, "Unknown firmware management setting")
	}
}

func (s *collectionService) CreateCollection(ctx context.Context, req *apipb.Collection) (*apipb.Collection, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "Missing collection parameters")
	}

	auth, err := s.EnsureAuth(ctx)
	if err != nil {
		return nil, err
	}

	collection := model.NewCollection()
	collection.TeamID = auth.User.PrivateTeamID
	collection.FieldMask = s.fieldMask.DefaultFields()
	if req.TeamId != nil {
		team, err := apitoolbox.EnsureTeamAdmin(auth.User.ID, req.TeamId.Value, s.store)
		if err != nil {
			return nil, err
		}
		collection.TeamID = team.ID
	}
	if req.Firmware != nil {
		newManagement, err := s.getFirmwareManagement(req.Firmware)
		if err != nil {
			return nil, err
		}
		collection.Firmware.Management = newManagement
	}

	if req.FieldMask != nil {
		apitoolbox.SetFieldMask(&collection.FieldMask, req.FieldMask, s.fieldMask)
	}
	// Assign any tags included with the request
	for k, v := range req.Tags {
		if !collection.IsValidTag(k, v) {
			return nil, status.Error(codes.InvalidArgument, "Invalid tag/name combination")
		}
		collection.SetTag(k, v)
	}
	// Note that any other fields are ignored if they're supplied. If f.e. the collection id
	// is set in the request it will be ignore.

	// Now store the new collection.
	collection.ID = s.store.NewCollectionID()
	if err := s.store.CreateCollection(auth.User.ID, collection); err != nil {
		logging.Warning("Error storing new collection for user %d: %v", auth.User.ID, err)
		return nil, status.Error(codes.Internal, "Unable to store the collection")
	}
	return apitoolbox.NewCollectionFromModel(collection), nil
}

func (s *collectionService) RetrieveCollection(ctx context.Context, req *apipb.RetrieveCollectionRequest) (*apipb.Collection, error) {
	if req == nil || req.CollectionId == nil {
		return nil, status.Error(codes.InvalidArgument, "Missing collection ID")
	}
	auth, err := s.EnsureAuth(ctx)
	if err != nil {
		return nil, err
	}
	coll, err := s.loadCollection(auth, req.CollectionId.Value)
	if err != nil {
		return nil, err
	}
	return apitoolbox.NewCollectionFromModel(coll), nil
}

func (s *collectionService) DeleteCollection(ctx context.Context, req *apipb.DeleteCollectionRequest) (*apipb.Collection, error) {
	if req == nil || req.CollectionId == nil {
		return nil, status.Error(codes.InvalidArgument, "Missing collection ID")
	}
	auth, err := s.EnsureAuth(ctx)
	if err != nil {
		return nil, err
	}
	coll, err := s.loadCollection(auth, req.CollectionId.Value)
	if err != nil {
		return nil, err
	}
	if err := s.store.DeleteCollection(auth.User.ID, coll.ID); err != nil {
		// There shouldn't be a not found error here unless there's two concurrent requests.
		logging.Warning("Unable to remove collection %d: %v", coll.ID, err)
		return nil, status.Error(codes.Internal, "Unable to remove collection")
	}
	return apitoolbox.NewCollectionFromModel(coll), nil
}

func (s *collectionService) UpdateCollection(ctx context.Context, req *apipb.Collection) (*apipb.Collection, error) {
	if req == nil || req.CollectionId == nil {
		return nil, status.Error(codes.InvalidArgument, "Missing collection ID")
	}
	auth, err := s.EnsureAuth(ctx)
	if err != nil {
		return nil, err
	}
	coll, err := s.loadCollection(auth, req.CollectionId.Value)
	if err != nil {
		return nil, err
	}

	// If the team ID is set in the request and different from the old make sure
	// the user is an admin of the new team
	if req.TeamId != nil {
		team, err := apitoolbox.EnsureTeamAdmin(auth.User.ID, req.TeamId.Value, s.store)
		if err != nil {
			return nil, err
		}
		coll.TeamID = team.ID
	}

	if req.Tags != nil {
		for k, v := range req.Tags {
			if !coll.IsValidTag(k, v) {
				return nil, status.Error(codes.InvalidArgument, "Invalid name/value for tags")
			}
			coll.SetTag(k, v)
		}
	}

	if req.Firmware != nil {
		checkFirmware := false
		if req.Firmware.CurrentFirmwareId != nil {
			// Check if firmware ID is valid, check if it exists, assign
			coll.Firmware.CurrentFirmwareID, err = model.NewFirmwareKeyFromString(req.Firmware.CurrentFirmwareId.Value)
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, "Invalid current firmware ID")
			}
			checkFirmware = true
		}
		if req.Firmware.TargetFirmwareId != nil {
			coll.Firmware.TargetFirmwareID, err = model.NewFirmwareKeyFromString(req.Firmware.TargetFirmwareId.Value)
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, "Invalid target firmware ID")
			}
			checkFirmware = true
		}
		if checkFirmware {
			_, _, err = s.store.RetrieveCurrentAndTargetFirmware(coll.ID, coll.Firmware.CurrentFirmwareID, coll.Firmware.TargetFirmwareID)
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, "Unknown firmware ID")
			}
		}
		if req.Firmware.Management != apipb.CollectionFirmware_unspecified {
			newManagement, err := s.getFirmwareManagement(req.Firmware)
			if err != nil {
				return nil, err
			}
			coll.Firmware.Management = newManagement
		}
	}

	fieldMaskChange := false
	if req.FieldMask != nil {
		fieldMaskChange = apitoolbox.SetFieldMask(&coll.FieldMask, req.FieldMask, s.fieldMask)
	}

	if err := s.store.UpdateCollection(auth.User.ID, coll); err != nil {
		logging.Warning("Error updating collection %d: %v", coll.ID, err)
		return nil, status.Error(codes.Internal, "Error updating collection")
	}

	if fieldMaskChange {
		outputs, err := s.store.ListOutputs(auth.User.ID, coll.ID)
		if err == nil {
			for _, v := range outputs {
				if err := s.outputManager.Update(v, s.fieldMask.ForcedFields()); err != nil {
					// Log error and continue
					logging.Warning("Error updating field mask on output %d: %v", v.ID, err)
				}
			}
		} else {
			// Just log error and continue
			logging.Warning("Unable to retrieve list of outputs for collection %d: %v", coll.ID, err)
		}
	}
	return apitoolbox.NewCollectionFromModel(coll), nil
}

func (s *collectionService) ListCollections(ctx context.Context, req *apipb.ListCollectionRequest) (*apipb.ListCollectionResponse, error) {
	if req == nil {
		// Technically not an issue but... let's be consistent
		return nil, status.Error(codes.InvalidArgument, "Missing request object")
	}
	auth, err := s.EnsureAuth(ctx)
	if err != nil {
		return nil, err
	}
	list, err := s.store.ListCollections(auth.User.ID)
	if err != nil {
		return nil, status.Error(codes.Internal, "Unable to list collections")
	}

	ret := &apipb.ListCollectionResponse{
		Collections: make([]*apipb.Collection, 0),
	}
	for _, v := range list {
		ret.Collections = append(ret.Collections, apitoolbox.NewCollectionFromModel(v))
	}
	return ret, nil
}

func (s *collectionService) ListCollectionMessages(ctx context.Context, req *apipb.ListMessagesRequest) (*apipb.ListMessagesResponse, error) {
	if req == nil || req.CollectionId == nil {
		return nil, status.Error(codes.InvalidArgument, "Missing collection ID")
	}
	auth, err := s.EnsureAuth(ctx)
	if err != nil {
		return nil, err
	}

	collection, err := s.loadCollection(auth, req.CollectionId.Value)
	if err != nil {
		return nil, err
	}

	dataFilter := &datastore.DataFilter{
		CollectionId: req.CollectionId.Value,
	}
	apitoolbox.ApplyDataFilter(req, dataFilter)

	result, err := s.dataStoreClient.GetData(ctx, dataFilter)
	if err != nil {
		logging.Warning("Error retrieving data from data store for collection %d: %v", collection.ID, err)
		return nil, status.Error(codes.Internal, "Error loading data from store")
	}

	ret := &apipb.ListMessagesResponse{}
	for {
		var msg datastore.DataMessage
		if err := result.RecvMsg(&msg); err != nil {
			result.CloseSend()
			break
		}

		dataMessage, err := apitoolbox.UnmarshalDataStoreMetadata(msg.Metadata, collection.FieldMask, msg.Payload, msg.Created)
		if err != nil {
			logging.Warning("Error unmarshaling detadata: %v", err)
			continue
		}
		ret.Messages = append(ret.Messages, dataMessage)
	}
	return ret, nil
}

func (s *collectionService) BroadcastMessage(ctx context.Context, req *apipb.SendMessageRequest) (*apipb.MultiSendMessageResponse, error) {
	if req == nil || req.CollectionId == nil {
		return nil, status.Error(codes.InvalidArgument, "Missing collection ID")
	}
	auth, err := s.EnsureAuth(ctx)
	if err != nil {
		return nil, err
	}

	collection, err := s.loadCollection(auth, req.CollectionId.Value)
	if err != nil {
		return nil, err
	}

	msg, err := apitoolbox.NewDownstreamMessage(req)
	if err != nil {
		return nil, err
	}

	devices, err := s.store.ListDevices(auth.User.ID, collection.ID)
	if err != nil {
		logging.Warning("Unable to retrieve device list for collection %d: %v", collection.ID, err)
		return nil, status.Error(codes.Internal, "Error loading device list")
	}

	res := &apipb.MultiSendMessageResponse{
		Errors: make([]*apipb.MessageSendResult, 0),
	}
	for _, v := range devices {
		if err := s.messageSender.Send(v, msg); err != nil {
			res.Errors = append(res.Errors, &apipb.MessageSendResult{
				DeviceId: &wrappers.StringValue{Value: v.ID.String()},
				Message:  &wrappers.StringValue{Value: err.Error()},
			})
			res.Failed++
			continue
		}
		res.Sent++
	}
	return res, nil
}

func (s *collectionService) MessageStream(req *apipb.MessageStreamRequest, svr apipb.Horde_MessageStreamServer) error {
	if req == nil || req.CollectionId == nil {
		return status.Error(codes.InvalidArgument, "Missing collection ID")
	}
	auth, err := s.EnsureAuth(svr.Context())
	if err != nil {
		return err
	}
	coll, err := s.loadCollection(auth, req.CollectionId.Value)
	if err != nil {
		return err
	}

	deviceID := model.DeviceKey(0)
	if req.DeviceId != nil {
		deviceID, err = model.NewDeviceKeyFromString(req.DeviceId.Value)
		if err != nil {
			return status.Error(codes.InvalidArgument, "Invalid device ID")
		}
	}
	go func() {
		ch := s.outputManager.Subscribe(coll.ID)

		defer s.outputManager.Unsubscribe(ch)
		for msg := range ch {
			data, ok := msg.(model.DataMessage)
			if !ok {
				logging.Error("Did not get model.DataMessage from channel. Got %T (%+v)", msg, msg)
				continue
			}
			if deviceID == 0 || deviceID == data.Device.ID {
				if err := svr.Send(apitoolbox.NewOutputDataMessageFromModel(data, coll)); err != nil {
					logging.Debug("Got error %v sending data message to client", err)
					return
				}
			}
		}
	}()
	return nil
}

// Tag implementation

func (s *collectionService) LoadTaggedResource(auth *authResult, collectionID, identifier string) (taggedResource, error) {
	coll, err := s.loadCollection(auth, collectionID)
	if err != nil {
		return nil, err
	}
	return &coll, nil
}

func (s *collectionService) loadCollection(auth *authResult, identifier string) (model.Collection, error) {
	collectionID, err := model.NewCollectionKeyFromString(identifier)
	if err != nil {
		return model.Collection{}, status.Error(codes.InvalidArgument, "Invalid collection ID")
	}
	coll, err := s.store.RetrieveCollection(auth.User.ID, collectionID)
	if err != nil {
		if err == storage.ErrNotFound {
			return model.Collection{}, status.Error(codes.NotFound, "Unknown collection")
		}
		logging.Warning("Error retrieving collection %d: %v", collectionID, err)
		return model.Collection{}, status.Error(codes.Internal, "Unable to retrieve collection")
	}
	return coll, nil
}

func (s *collectionService) UpdateResourceTags(id model.UserKey, collectionID, identifier string, res interface{}) error {
	coll := res.(*model.Collection)
	return s.store.UpdateCollectionTags(id, collectionID, coll.Tags)
}

func (s *collectionService) ListCollectionTags(ctx context.Context, req *apipb.TagRequest) (*apipb.TagResponse, error) {
	return listTags(ctx, req, s)
}

func (s *collectionService) UpdateCollectionTags(ctx context.Context, req *apipb.UpdateTagRequest) (*apipb.TagResponse, error) {
	return updateTags(ctx, req, s)
}

func (s *collectionService) GetCollectionTag(ctx context.Context, req *apipb.TagRequest) (*apipb.TagValueResponse, error) {
	return getTag(ctx, req, s)
}

func (s *collectionService) DeleteCollectionTag(ctx context.Context, req *apipb.TagRequest) (*apipb.TagValueResponse, error) {
	return deleteTag(ctx, req, s)
}

func (s *collectionService) UpdateCollectionTag(ctx context.Context, req *apipb.TagRequest) (*apipb.TagValueResponse, error) {
	return updateTag(ctx, req, s)
}
