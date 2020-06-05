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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/eesrc/horde/pkg/api/apipb"
	"github.com/eesrc/horde/pkg/api/apitoolbox"
	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/output"
	"github.com/eesrc/horde/pkg/storage"
)

// newOutputService creates the server-side object for output resources under a collection
func newOutputService(
	store storage.DataStore,
	manager output.Manager,
	fieldMask model.FieldMaskParameters) outputService {
	return outputService{
		store:           store,
		manager:         manager,
		fieldMask:       fieldMask,
		defaultGrpcAuth: defaultGrpcAuth{Store: store},
	}
}

type outputService struct {
	store     storage.DataStore
	manager   output.Manager
	fieldMask model.FieldMaskParameters

	defaultGrpcAuth
}

func (s *outputService) loadOutput(auth *authResult, collectionID, outputID string) (model.Output, error) {
	cID, err := model.NewCollectionKeyFromString(collectionID)
	if err != nil {
		return model.Output{}, status.Error(codes.InvalidArgument, "Invalid collection ID")
	}
	oID, err := model.NewOutputKeyFromString(outputID)
	if err != nil {
		return model.Output{}, status.Error(codes.InvalidArgument, "Invalid output ID")
	}
	output, err := s.store.RetrieveOutput(auth.User.ID, cID, oID)
	if err != nil {
		if err == storage.ErrNotFound {
			return model.Output{}, status.Error(codes.NotFound, "Unknown output")
		}
		logging.Warning("Unable to retrieve output %d (collection ID = %d): %v", oID, cID, err)
		return model.Output{}, status.Error(codes.Internal, "Unable to retrieve output")
	}
	return output, nil
}

func (s *outputService) RetrieveOutput(ctx context.Context, req *apipb.OutputRequest) (*apipb.Output, error) {
	if req == nil || req.CollectionId == nil || req.OutputId == nil {
		return nil, status.Error(codes.InvalidArgument, "Missing request object")
	}
	auth, err := s.EnsureAuth(ctx)
	if err != nil {
		return nil, err
	}

	output, err := s.loadOutput(auth, req.CollectionId.Value, req.OutputId.Value)
	if err != nil {
		return nil, err
	}
	return apitoolbox.NewOutputFromModel(output), nil
}

func (s *outputService) DeleteOutput(ctx context.Context, req *apipb.OutputRequest) (*apipb.Output, error) {
	if req == nil || req.CollectionId == nil || req.OutputId == nil {
		return nil, status.Error(codes.InvalidArgument, "Missing request object")
	}
	auth, err := s.EnsureAuth(ctx)
	if err != nil {
		return nil, err
	}

	output, err := s.loadOutput(auth, req.CollectionId.Value, req.OutputId.Value)
	if err != nil {
		return nil, err
	}
	if err := s.store.DeleteOutput(auth.User.ID, output.CollectionID, output.ID); err != nil {
		if err == storage.ErrAccess {
			return nil, status.Error(codes.PermissionDenied, "Must be administrator to remove output")
		}
		logging.Warning("Unable to remove output %d (collection ID = %d): %v", output.ID, output.CollectionID, err)
		return nil, status.Error(codes.Internal, "Unable to delete output")
	}
	// Stop the output if it is running
	if err := s.manager.Stop(output.ID); err != nil {
		logging.Warning("Unable to stop output %d: %v", output.ID, err)
	}
	return apitoolbox.NewOutputFromModel(output), nil
}

// Stubs below

func (s *outputService) CreateOutput(ctx context.Context, req *apipb.Output) (*apipb.Output, error) {
	if req == nil || req.CollectionId == nil {
		return nil, status.Error(codes.InvalidArgument, "Missing request object")
	}
	if req.Config == nil {
		return nil, status.Error(codes.InvalidArgument, "Must specify a configuration")
	}
	auth, err := s.EnsureAuth(ctx)
	if err != nil {
		return nil, err
	}
	collectionID, err := model.NewCollectionKeyFromString(req.CollectionId.Value)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid collection ID")
	}

	newOutput := model.NewOutput()
	newOutput.Type = req.Type.String()
	newOutput.Config = apitoolbox.NewOutputConfigFromAPI(req)
	newOutput.ID = s.store.NewOutputID()
	newOutput.CollectionID = collectionID
	if req.Tags != nil {
		for k, v := range req.Tags {
			if !newOutput.IsValidTag(k, v) {
				return nil, status.Error(codes.InvalidArgument, "Invalid tag/name combination")
			}
			newOutput.SetTag(k, v)
		}
	}

	newOutput.Enabled = true

	if req.Enabled != nil {
		newOutput.Enabled = req.Enabled.Value
	}
	if messages, err := s.manager.Verify(newOutput); err != nil {
		s, err := status.New(codes.InvalidArgument, "Output configuration is invalid").WithDetails(&apipb.ErrorDetails{Messages: messages})
		if err != nil {
			logging.Error("Could not send error: %v", err)
			return nil, status.Error(codes.InvalidArgument, "Config error")
		}
		return nil, s.Err()
	}
	if err := s.store.CreateOutput(auth.User.ID, newOutput); err != nil {
		if err == storage.ErrNotFound {
			return nil, status.Error(codes.NotFound, "Unknown collection")
		}
		if err == storage.ErrAccess {
			return nil, status.Error(codes.PermissionDenied, "Must be administrator to create outputs")
		}
		logging.Warning("Got error creating output on collection %d: %v", collectionID, err)
		return nil, status.Error(codes.Internal, "Unable to create output")
	}
	if newOutput.Enabled {
		if err := s.manager.Update(newOutput, s.fieldMask.ForcedFields()); err != nil {
			logging.Warning("Unable to update new output (id=%d collection id=%d): %v", newOutput.ID, newOutput.CollectionID, err)
			return nil, status.Error(codes.Internal, "Unable to start output")
		}
	}
	return apitoolbox.NewOutputFromModel(newOutput), nil
}

func (s *outputService) UpdateOutput(ctx context.Context, req *apipb.Output) (*apipb.Output, error) {
	if req == nil || req.CollectionId == nil || req.OutputId == nil {
		return nil, status.Error(codes.InvalidArgument, "Missing request object")
	}
	auth, err := s.EnsureAuth(ctx)
	if err != nil {
		return nil, err
	}
	output, err := s.loadOutput(auth, req.CollectionId.Value, req.OutputId.Value)
	if err != nil {
		return nil, err
	}
	update := false
	if req.Enabled != nil {
		if req.Enabled.Value != output.Enabled {
			output.Enabled = req.Enabled.Value
			update = true
		}
	}
	if req.Type != apipb.Output_undefined && req.Type.String() != output.Type {
		output.Type = req.Type.String()
		output.Config = make(model.OutputConfig)
		update = true
	}

	if req.Config != nil {
		output.Config = apitoolbox.NewOutputConfigFromAPI(req)
		update = true
	}
	if messages, err := s.manager.Verify(output); err != nil {
		s, err := status.New(codes.InvalidArgument, "Output configuration is invalid").WithDetails(&apipb.ErrorDetails{Messages: messages})
		if err != nil {
			logging.Error("Could not send error: %v", err)
			return nil, status.Error(codes.InvalidArgument, "Config error")
		}
		return nil, s.Err()
	}
	if req.Tags != nil {
		for k, v := range req.Tags {
			if !output.IsValidTag(k, v) {
				return nil, status.Error(codes.InvalidArgument, "Invalid tag name/value")
			}
			output.SetTag(k, v)
		}
		update = true
	}
	if update {
		if err := s.store.UpdateOutput(auth.User.ID, output.CollectionID, output); err != nil {
			// There *might* be a NotFound error here if the collection or output is
			// removed between the retrieval above and this update but it's
			// a corner case. We'll get the error in the logs so if it becomes
			// a problem we'll handle it properly via ETags or similar.
			if err == storage.ErrAccess {
				return nil, status.Error(codes.PermissionDenied, "Must be administrator to update collection")
			}
			logging.Warning("Unable to update output %d (collection ID = %d): %v", output.ID, output.CollectionID, err)
			return nil, status.Error(codes.Internal, "Unable to update output")
		}
		// Restart output regardless of config change.
		if err := s.manager.Stop(output.ID); err != nil {
			// Won't report back to the client since the output might be stopped or invalid.
			// Wait and see what happens when we update it.
			logging.Info("Got error stopping output %d (collection ID=%d): %v", output.ID, output.CollectionID, err)
		}
		// The output manager will check if the output is enabled or not before starting so
		// no need to check if we should start it.
		if err := s.manager.Update(output, s.fieldMask.ForcedFields()); err != nil {
			return nil, status.Error(codes.Internal, "Unable to update output")
		}
	}

	return apitoolbox.NewOutputFromModel(output), nil
}

func (s *outputService) ListOutputs(ctx context.Context, req *apipb.ListOutputRequest) (*apipb.ListOutputResponse, error) {
	if req == nil || req.CollectionId == nil {
		return nil, status.Error(codes.InvalidArgument, "Missing request object")
	}
	auth, err := s.EnsureAuth(ctx)
	if err != nil {
		return nil, err
	}

	collectionID, err := model.NewCollectionKeyFromString(req.CollectionId.Value)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid collection ID")
	}

	list, err := s.store.ListOutputs(auth.User.ID, collectionID)
	if err != nil {
		logging.Warning("Unable to list outputs for user %d: %v", auth.User.ID, err)
		return nil, status.Error(codes.Internal, "Unable to read list of outputs")
	}
	ret := &apipb.ListOutputResponse{
		Outputs: make([]*apipb.Output, 0),
	}
	for _, v := range list {
		ret.Outputs = append(ret.Outputs, apitoolbox.NewOutputFromModel(v))
	}
	return ret, nil
}

func (s *outputService) Logs(ctx context.Context, req *apipb.OutputRequest) (*apipb.OutputLogs, error) {
	if req == nil || req.CollectionId == nil || req.OutputId == nil {
		return nil, status.Error(codes.InvalidArgument, "Missing request object")
	}
	auth, err := s.EnsureAuth(ctx)
	if err != nil {
		return nil, err
	}
	output, err := s.loadOutput(auth, req.CollectionId.Value, req.OutputId.Value)
	if err != nil {
		return nil, err
	}
	op, err := s.manager.Get(output.ID)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, "Output isn't running")
	}

	return &apipb.OutputLogs{
		Logs: apitoolbox.NewOutputLogsFromModel(op.Logs()),
	}, nil
}

func (s *outputService) Status(ctx context.Context, req *apipb.OutputRequest) (*apipb.OutputStatus, error) {
	if req == nil || req.CollectionId == nil || req.OutputId == nil {
		return nil, status.Error(codes.InvalidArgument, "Missing request object")
	}
	auth, err := s.EnsureAuth(ctx)
	if err != nil {
		return nil, err
	}
	output, err := s.loadOutput(auth, req.CollectionId.Value, req.OutputId.Value)
	if err != nil {
		return nil, err
	}
	op, err := s.manager.Get(output.ID)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, "Output isn't running")
	}
	return apitoolbox.NewOutputStatusFromModel(output.CollectionID.String(), output.ID.String(), output.Enabled, op.Status()), nil
}

// Tag implementation. This is going to be a bit different since it uses both
// a collection ID and a device ID to retrieve the device as opposed to the
// team/collection/token tag updates that uses a single identifier for the
// resource.
func (s *outputService) LoadTaggedResource(auth *authResult, collectionID string, outputID string) (taggedResource, error) {
	o, err := s.loadOutput(auth, collectionID, outputID)
	if err != nil {
		return nil, err
	}
	return &o, err
}

func (s *outputService) UpdateResourceTags(id model.UserKey, collectionID, identifier string, res interface{}) error {
	o := res.(*model.Output)
	return s.store.UpdateOutputTags(id, identifier, o.Tags)
}

func (s *outputService) ListOutputTags(ctx context.Context, req *apipb.TagRequest) (*apipb.TagResponse, error) {
	return listTags(ctx, req, s)
}

func (s *outputService) UpdateOutputTags(ctx context.Context, req *apipb.UpdateTagRequest) (*apipb.TagResponse, error) {
	return updateTags(ctx, req, s)
}

func (s *outputService) GetOutputTag(ctx context.Context, req *apipb.TagRequest) (*apipb.TagValueResponse, error) {
	return getTag(ctx, req, s)
}

func (s *outputService) DeleteOutputTag(ctx context.Context, req *apipb.TagRequest) (*apipb.TagValueResponse, error) {
	return deleteTag(ctx, req, s)
}

func (s *outputService) UpdateOutputTag(ctx context.Context, req *apipb.TagRequest) (*apipb.TagValueResponse, error) {
	return updateTag(ctx, req, s)
}
