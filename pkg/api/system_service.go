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
	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/grpc/status"

	"github.com/eesrc/horde/pkg/addons/magpie/datastore"
	"github.com/eesrc/horde/pkg/api/apipb"
	"github.com/eesrc/horde/pkg/api/apitoolbox"
	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
	"github.com/eesrc/horde/pkg/version"
	"google.golang.org/grpc/codes"
)

// systemService is the system service for the API. There are multiple services
// right now but they will be merged into a single API service later
type systemService struct {
	fieldMask       model.FieldMaskParameters
	store           storage.DataStore
	deviceDataStore datastore.DataStoreClient
}

// newSystemService creates a new system service instance.
func newSystemService(
	fieldMask model.FieldMaskParameters,
	store storage.DataStore,
	dataStore datastore.DataStoreClient) systemService {
	return systemService{
		fieldMask:       fieldMask,
		store:           store,
		deviceDataStore: dataStore,
	}
}

func (s *systemService) DataDump(ctx context.Context, req *apipb.DataDumpRequest) (*apipb.DataDumpResponse, error) {
	// Check authentication
	auth := gRPCAuth(ctx, s.store)
	if auth == nil {
		return nil, status.Error(codes.Unauthenticated, "Must authenticate")
	}

	ret := &apipb.DataDumpResponse{
		Profile: apitoolbox.NewUserProfileFromUser(auth.User),
	}
	teams, err := s.store.ListTeams(auth.User.ID)
	if err != nil {
		logging.Warning("Unable to read teams for user %d: %v", auth.User.ID, err)
		return nil, status.Error(codes.Internal, "Error reading team list")
	}
	ret.Teams = make([]*apipb.Team, 0)
	for _, v := range teams {
		ret.Teams = append(ret.Teams, apitoolbox.NewTeamFromModel(v, true))
	}

	tokens, err := s.store.ListTokens(auth.User.ID)
	if err != nil {
		logging.Warning("Unable to read tokens for user %d: %v", auth.User.ID, err)
		return nil, status.Error(codes.Internal, "Error reading token list")
	}
	ret.Tokens = make([]*apipb.Token, 0)
	for _, v := range tokens {
		ret.Tokens = append(ret.Tokens, apitoolbox.NewTokenFromModel(v))
	}

	collections, err := s.store.ListCollections(auth.User.ID)
	if err != nil {
		logging.Warning("Unable to read collections for user: %d: %v", auth.User.ID, err)
		return nil, status.Error(codes.Internal, "Error reading collection list")
	}

	ret.Collections = make([]*apipb.DumpedCollection, 0)
	for _, v := range collections {
		coll, err := s.readCollection(ctx, auth.User, v)
		if err != nil {
			logging.Warning("Unable to read contents of collection %d: %v", v.ID, err)
			return nil, status.Error(codes.Internal, "Error reading contents of collection")
		}
		ret.Collections = append(ret.Collections, coll)
	}
	return ret, nil
}

// Read devices, device data and outputs from collection
func (s *systemService) readCollection(ctx context.Context, user model.User, c model.Collection) (*apipb.DumpedCollection, error) {
	ret := &apipb.DumpedCollection{}
	ret.Collection = apitoolbox.NewCollectionFromModel(c)

	devices, err := s.store.ListDevices(user.ID, c.ID)
	if err != nil {
		return nil, err
	}
	ret.Devices = make([]*apipb.DumpedDevice, 0)
	for _, v := range devices {
		device := apitoolbox.NewDeviceFromModel(v, c)
		data, err := s.loadDataForDevice(ctx, c.ID, c.FieldMask, v.ID)
		if err != nil {
			logging.Warning("Unable to load data for device %d in collection %d: %v. Skipping device.", v.ID, c.ID, err)
			continue
		}
		dd := &apipb.DumpedDevice{
			Device: device,
			Data:   data,
		}
		ret.Devices = append(ret.Devices, dd)
	}
	outputs, err := s.store.ListOutputs(user.ID, c.ID)
	if err != nil {
		return nil, err
	}
	ret.Outputs = make([]*apipb.Output, 0)
	for _, v := range outputs {
		output := apitoolbox.NewOutputFromModel(v)
		ret.Outputs = append(ret.Outputs, output)
	}
	return ret, nil
}

func (s *systemService) loadDataForDevice(ctx context.Context, collectionID model.CollectionKey, fieldMask model.FieldMask, deviceID model.DeviceKey) ([]*apipb.OutputDataMessage, error) {
	stream, err := s.deviceDataStore.GetData(ctx, &datastore.DataFilter{
		CollectionId: collectionID.String(),
		DeviceId:     deviceID.String(),
	})
	if err != nil {
		return nil, err
	}

	ret := make([]*apipb.OutputDataMessage, 0)
	for {
		msg, err := stream.Recv()
		if err != nil {
			return ret, nil
		}
		dataMessage, err := apitoolbox.UnmarshalDataStoreMetadata(msg.Metadata, fieldMask, msg.Payload, msg.Created)
		if err != nil {
			continue
		}
		ret = append(ret, dataMessage)
	}
}

func (s *systemService) GetSystemInfo(ctx context.Context, req *apipb.SystemInfoRequest) (*apipb.SystemInfoResponse, error) {
	ret := &apipb.SystemInfoResponse{
		Version:          &wrappers.StringValue{Value: version.Number},
		BuildDate:        &wrappers.StringValue{Value: version.BuildDate},
		ReleaseName:      &wrappers.StringValue{Value: version.Name},
		DefaultFieldMask: apitoolbox.NewFieldMaskFromModel(s.fieldMask.DefaultFields()),
		ForcedFieldMask:  apitoolbox.NewFieldMaskFromModel(s.fieldMask.ForcedFields()),
	}
	return ret, nil
}

func (s *systemService) GetUserProfile(ctx context.Context, req *apipb.UserProfileRequest) (*apipb.UserProfile, error) {
	auth := gRPCAuth(ctx, s.store)
	if auth == nil {
		return nil, status.Error(codes.Unauthenticated, "Must authenticate")
	}

	return apitoolbox.NewUserProfileFromUser(auth.User), nil
}
