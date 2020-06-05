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
	"strings"

	"github.com/ExploratoryEngineering/logging"
	"github.com/eesrc/horde/pkg/api/apipb"
	"github.com/eesrc/horde/pkg/model"
	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Tag implementation for the services. This is common behaviour for several
// resources.
type taggedResource interface {
	TagData() map[string]string
	IsValidTag(k, v string) bool
	SetTag(k, v string)
	GetTag(k string) string
}

type taggedService interface {
	// Ensure user is authenticated. If it's a valid authentication authResult
	// is returned, otherwise an error is returned. The error is returned as
	// is.
	EnsureAuth(ctx context.Context) (*authResult, error)

	// Load a resource with tags. If there's an error an error is returned. The
	// error is returned as is.
	LoadTaggedResource(auth *authResult, collectionID string, identifier string) (taggedResource, error)

	UpdateResourceTags(id model.UserKey, collectionID string, identifier string, res interface{}) error
}

func listTags(ctx context.Context, req *apipb.TagRequest, svc taggedService) (*apipb.TagResponse, error) {
	if req == nil || (req.CollectionId == nil && req.Identifier == nil) {
		return nil, status.Error(codes.InvalidArgument, "Missing identifier")
	}
	auth, err := svc.EnsureAuth(ctx)
	if err != nil {
		return nil, err
	}
	cid := ""
	if req.CollectionId != nil {
		cid = req.CollectionId.Value
	}
	val := ""
	if req.Identifier != nil {
		val = req.Identifier.Value
	}
	res, err := svc.LoadTaggedResource(auth, cid, val)
	if err != nil {
		return nil, err
	}
	return &apipb.TagResponse{
		Tags: res.TagData(),
	}, nil
}

func updateTags(ctx context.Context, req *apipb.UpdateTagRequest, svc taggedService) (*apipb.TagResponse, error) {
	if req == nil || (req.CollectionId == nil && req.Identifier == nil) || req.Tags == nil {
		return nil, status.Error(codes.InvalidArgument, "Missing request")
	}
	auth, err := svc.EnsureAuth(ctx)
	if err != nil {
		return nil, err
	}
	cid := ""
	if req.CollectionId != nil {
		cid = req.CollectionId.Value
	}
	val := ""
	if req.Identifier != nil {
		val = req.Identifier.Value
	}
	res, err := svc.LoadTaggedResource(auth, cid, val)
	if err != nil {
		return nil, err
	}
	for k, v := range req.Tags {
		if !res.IsValidTag(k, v) {
			return nil, status.Error(codes.InvalidArgument, "Invalid key/value for tag")
		}
		res.SetTag(k, v)
	}
	if err := svc.UpdateResourceTags(auth.User.ID, cid, val, res); err != nil {
		// This *could* return not found but the resource would have to be
		// deleted after it's been retrieved above. Log and return error
		logging.Warning("Got error updating tag: %v", err)
		return nil, status.Error(codes.Internal, "Could not update tags")
	}
	return &apipb.TagResponse{Tags: res.TagData()}, nil
}

func getTag(ctx context.Context, req *apipb.TagRequest, svc taggedService) (*apipb.TagValueResponse, error) {
	if req == nil || (req.CollectionId == nil && req.Identifier == nil) || req.Name == nil || strings.TrimSpace(req.Name.Value) == "" {
		return nil, status.Error(codes.InvalidArgument, "Missing name")
	}
	auth, err := svc.EnsureAuth(ctx)
	if err != nil {
		return nil, err
	}
	cid := ""
	if req.CollectionId != nil {
		cid = req.CollectionId.Value
	}
	val := ""
	if req.Identifier != nil {
		val = req.Identifier.Value
	}
	res, err := svc.LoadTaggedResource(auth, cid, val)
	if err != nil {
		return nil, err
	}
	return &apipb.TagValueResponse{Value: &wrappers.StringValue{
		Value: res.GetTag(req.Name.Value)},
	}, nil
}

func deleteTag(ctx context.Context, req *apipb.TagRequest, svc taggedService) (*apipb.TagValueResponse, error) {
	if req == nil || (req.CollectionId == nil && req.Identifier == nil) {
		return nil, status.Error(codes.InvalidArgument, "Missing request")
	}
	req.Value = &wrappers.StringValue{Value: ""}

	return updateTag(ctx, req, svc)
}

func updateTag(ctx context.Context, req *apipb.TagRequest, svc taggedService) (*apipb.TagValueResponse, error) {
	if req == nil || (req.CollectionId == nil && req.Identifier == nil) || req.Name == nil || req.Value == nil {
		return nil, status.Error(codes.InvalidArgument, "Missing name/value pair")
	}
	auth, err := svc.EnsureAuth(ctx)
	if err != nil {
		return nil, err
	}
	cid := ""
	if req.CollectionId != nil {
		cid = req.CollectionId.Value
	}
	val := ""
	if req.Identifier != nil {
		val = req.Identifier.Value
	}
	res, err := svc.LoadTaggedResource(auth, cid, val)
	if err != nil {
		return nil, err
	}
	newVal := strings.TrimSpace(req.Value.Value)
	if !res.IsValidTag(strings.TrimSpace(req.Name.Value), newVal) {
		return nil, status.Error(codes.InvalidArgument, "Invalid key/value for tag")
	}
	res.SetTag(req.Name.Value, newVal)
	if err := svc.UpdateResourceTags(auth.User.ID, cid, val, res); err != nil {
		// This *could* return not found if the resource is deleted after it's been loaded
		// above but it shouldn't happen.
		logging.Warning("Got error updating tag: %v", err)
		return nil, status.Error(codes.Internal, "Could not update tag")
	}
	return &apipb.TagValueResponse{Value: &wrappers.StringValue{Value: newVal}}, nil
}
