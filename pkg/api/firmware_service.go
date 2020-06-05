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
	"bytes"
	"context"
	"strings"
	"time"

	"github.com/ExploratoryEngineering/logging"
	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/grpc/codes"

	"github.com/eesrc/horde/pkg/api/apipb"
	"github.com/eesrc/horde/pkg/api/apitoolbox"
	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
	"google.golang.org/grpc/status"
)

// newFirmwareService creates the server-side component of the firmware API. This
// server handles the server-side updates. The majority of methods are exposed via
// the grpc gateway library but some are used by custom HTTP handlers that
// handle the actual file uploads.
func newFirmwareService(store storage.DataStore, imageStore storage.FirmwareImageStore) firmwareService {
	return firmwareService{
		store:           store,
		imageStore:      imageStore,
		defaultGrpcAuth: defaultGrpcAuth{Store: store},
	}
}

type firmwareService struct {
	store storage.DataStore
	defaultGrpcAuth
	imageStore storage.FirmwareImageStore
}

func (fs *firmwareService) loadFirmware(auth *authResult, collectionID, firmwareID string) (model.Firmware, error) {
	cID, err := model.NewCollectionKeyFromString(collectionID)
	if err != nil {
		return model.Firmware{}, status.Error(codes.InvalidArgument, "Invalid collection ID")
	}
	fwID, err := model.NewFirmwareKeyFromString(firmwareID)
	if err != nil {
		return model.Firmware{}, status.Error(codes.InvalidArgument, "Invalid output ID")
	}
	firmware, err := fs.store.RetrieveFirmware(auth.User.ID, cID, fwID)
	if err != nil {
		if err == storage.ErrNotFound {
			return model.Firmware{}, status.Error(codes.NotFound, "Unknown firmware")
		}
		logging.Warning("Unable to retrieve firmware %d (collection ID = %d): %v", fwID, cID, err)
		return model.Firmware{}, status.Error(codes.Internal, "Unable to retrieve output")
	}
	return firmware, nil
}

func (fs *firmwareService) CreateFirmware(ctx context.Context, req *apipb.CreateFirmwareRequest) (*apipb.Firmware, error) {
	// Do quick verification on parameters before doing a roundtrip to
	// authenticate. Ideally this would be *after* the authentication but
	// it reduces the response time for invalid arguments.
	if req == nil || req.CollectionId == nil {
		return nil, status.Error(codes.InvalidArgument, "Missing collection ID")
	}
	if len(req.Image) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Image buffer required")
	}
	if req.Filename == nil || len(strings.TrimSpace(req.Filename.Value)) == 0 {
		return nil, status.Error(codes.InvalidArgument, "File name required")
	}
	if req.Version != nil && len(strings.TrimSpace(req.Version.Value)) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Version required")
	}
	collectionID, err := model.NewCollectionKeyFromString(req.CollectionId.Value)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid collection ID")
	}
	auth, err := fs.EnsureAuth(ctx)
	if err != nil {
		return nil, err
	}

	// Set versions and validate tags before we do a roundtrip to the
	// store.
	firmware := model.NewFirmware()
	firmware.Filename = req.Filename.Value
	for k, v := range req.Tags {
		if !firmware.IsValidTag(k, v) {
			return nil, status.Error(codes.InvalidArgument, "Invalid tag name/value")
		}
		firmware.SetTag(k, v)
	}
	firmware.ID = fs.store.NewFirmwareID()
	firmware.Created = time.Now()
	firmware.CollectionID = collectionID
	firmware.Length = len(req.Image)
	// Store the image in the store and get checksum
	cs, err := fs.imageStore.Create(firmware.ID, bytes.NewReader(req.Image))
	if err != nil {
		logging.Warning("Unable to create firmware image in image store: %v", err)
		return nil, status.Error(codes.Internal, "Unable to create image file")
	}
	firmware.SHA256 = cs
	version := cs
	if req.Version != nil {
		version = req.Version.Value
	}
	firmware.Version = version
	if err := fs.store.CreateFirmware(auth.User.ID, firmware); err != nil {
		if err := fs.imageStore.Delete(firmware.ID); err != nil {
			logging.Warning("Unable to remove image %d from image store: %v", firmware.ID, err)
		}
		if err == storage.ErrAccess {
			return nil, status.Error(codes.PermissionDenied, "Must be administrator to create firmware images")
		}
		if err == storage.ErrSHAAlreadyExists {
			return nil, status.Error(codes.FailedPrecondition, "An identical image already exists in this collection")
		}
		if err == storage.ErrAlreadyExists {
			return nil, status.Error(codes.FailedPrecondition, "A version with this number already exists in this collection")
		}
		logging.Warning("Unable to create firmware %d (collection id = %d): %v", firmware.ID, collectionID, err)
		return nil, status.Error(codes.Internal, "Unable to create image")
	}
	return apitoolbox.NewFirmwareFromModel(firmware), nil
}

func (fs *firmwareService) RetrieveFirmware(ctx context.Context, req *apipb.FirmwareRequest) (*apipb.Firmware, error) {
	if req == nil || req.CollectionId == nil || req.ImageId == nil {
		return nil, status.Error(codes.InvalidArgument, "Missing collection/firmware ID")
	}
	auth, err := fs.EnsureAuth(ctx)
	if err != nil {
		return nil, err
	}
	firmware, err := fs.loadFirmware(auth, req.CollectionId.Value, req.ImageId.Value)
	if err != nil {
		return nil, err
	}
	return apitoolbox.NewFirmwareFromModel(firmware), nil
}

func (fs *firmwareService) UpdateFirmware(ctx context.Context, req *apipb.Firmware) (*apipb.Firmware, error) {
	if req == nil || req.CollectionId == nil || req.ImageId == nil {
		return nil, status.Error(codes.InvalidArgument, "Missing collection/firmware ID")
	}
	auth, err := fs.EnsureAuth(ctx)
	if err != nil {
		return nil, err
	}
	firmware, err := fs.loadFirmware(auth, req.CollectionId.Value, req.ImageId.Value)
	if err != nil {
		return nil, err
	}
	// Return error if one of the read-only fields are modified
	if req.Filename != nil || req.Sha256 != nil || req.Length != nil {
		return nil, status.Error(codes.InvalidArgument, "Only version and tags can be modified for firmware images")
	}
	if req.Version == nil && req.Tags == nil {
		return nil, status.Error(codes.InvalidArgument, "Nothing to update")
	}
	if req.Version != nil {
		firmware.Version = strings.TrimSpace(req.Version.Value)
		if len(firmware.Version) == 0 {
			return nil, status.Error(codes.InvalidArgument, "Version can't be blank")
		}
	}
	if req.Tags != nil {
		for k, v := range req.Tags {
			if !firmware.IsValidTag(k, v) {
				return nil, status.Error(codes.InvalidArgument, "Invalid tag/name")
			}
			firmware.SetTag(k, v)
		}
	}
	if err := fs.store.UpdateFirmware(auth.User.ID, firmware.CollectionID, firmware); err != nil {
		if err == storage.ErrAccess {
			return nil, status.Error(codes.PermissionDenied, "Must be administrator to modify firmware")
		}
		logging.Warning("Unable to update firmware %d (collection id = %d): %v", firmware.ID, firmware.CollectionID, err)
		return nil, status.Error(codes.Internal, "Unable to update firmware")
	}
	return apitoolbox.NewFirmwareFromModel(firmware), nil
}

func (fs *firmwareService) DeleteFirmware(ctx context.Context, req *apipb.FirmwareRequest) (*apipb.Firmware, error) {
	if req == nil || req.CollectionId == nil || req.ImageId == nil {
		return nil, status.Error(codes.InvalidArgument, "Missing collection/firmware ID")
	}
	auth, err := fs.EnsureAuth(ctx)
	if err != nil {
		return nil, err
	}
	firmware, err := fs.loadFirmware(auth, req.CollectionId.Value, req.ImageId.Value)
	if err != nil {
		return nil, err
	}
	if err := fs.store.DeleteFirmware(auth.User.ID, firmware.CollectionID, firmware.ID); err != nil {
		if err == storage.ErrAccess {
			return nil, status.Error(codes.PermissionDenied, "Must be an administrator to remove firmware")
		}
		if err == storage.ErrReference {
			return nil, status.Error(codes.FailedPrecondition, "Firmware is in use")
		}
		// NotFound might be returned but the loadFirmware call above should return
		// it unless there's a race condition. We'll log it so if it becomes a problem
		// we'll solve it with ETags or similar mechanisms.
		logging.Warning("Unable to remove firmware %d (collection ID=%d): %v", firmware.ID, firmware.CollectionID, err)
		return nil, status.Error(codes.Internal, "Unable to remove firmware")
	}
	if err := fs.imageStore.Delete(firmware.ID); err != nil {
		// Just log the error. The image is removed from the store so it won't be
		// visible.
		logging.Error("Unable to remove image %d from store: %v", firmware.ID, err)
	}
	return apitoolbox.NewFirmwareFromModel(firmware), nil

}

func (fs *firmwareService) ListFirmware(ctx context.Context, req *apipb.ListFirmwareRequest) (*apipb.ListFirmwareResponse, error) {
	if req == nil || req.CollectionId == nil {
		return nil, status.Error(codes.InvalidArgument, "Missing collection ID")
	}
	collectionID, err := model.NewCollectionKeyFromString(req.CollectionId.Value)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid collection ID")
	}
	auth, err := fs.EnsureAuth(ctx)
	if err != nil {
		return nil, err
	}
	list, err := fs.store.ListFirmware(auth.User.ID, collectionID)
	if err != nil {
		logging.Warning("Unable to list firmware for collection %d: %v", collectionID, err)
		return nil, status.Error(codes.Internal, "Unable to read collection list")
	}

	ret := &apipb.ListFirmwareResponse{
		Images: make([]*apipb.Firmware, 0),
	}
	for _, v := range list {
		ret.Images = append(ret.Images, apitoolbox.NewFirmwareFromModel(v))
	}
	return ret, nil
}

func (fs *firmwareService) FirmwareUsage(ctx context.Context, req *apipb.FirmwareRequest) (*apipb.FirmwareUsageResponse, error) {
	if req == nil || req.CollectionId == nil || req.ImageId == nil {
		return nil, status.Error(codes.InvalidArgument, "Missing collection/firmware ID")
	}
	auth, err := fs.EnsureAuth(ctx)
	if err != nil {
		return nil, err
	}
	// Sanity check if the firmware really exists. This is an extra roundtrip
	// to the store but it makes the API behave consistent. Also it checks the
	// conversion for collection ID and firmware ID
	fw, err := fs.loadFirmware(auth, req.CollectionId.Value, req.ImageId.Value)
	if err != nil {
		return nil, err
	}
	use, err := fs.store.RetrieveFirmwareVersionsInUse(auth.User.ID, fw.CollectionID, fw.ID)
	if err != nil {
		logging.Warning("Unable to retrieve list of firmware images in use (firmwareID=%d collection ID=%d): %v", fw.ID, fw.CollectionID, err)
		return nil, status.Error(codes.Internal, "Unable to query firmware images")
	}
	ret := &apipb.FirmwareUsageResponse{
		ImageId:  &wrappers.StringValue{Value: use.FirmwareID.String()},
		Targeted: make([]string, 0),
		Current:  make([]string, 0),
	}
	for _, v := range use.Targeted {
		ret.Targeted = append(ret.Targeted, v.String())
	}
	for _, v := range use.Current {
		ret.Current = append(ret.Current, v.String())
	}
	return ret, nil
}

// Tag implementation
func (fs *firmwareService) LoadTaggedResource(auth *authResult, collectionID string, firmwareID string) (taggedResource, error) {
	fw, err := fs.loadFirmware(auth, collectionID, firmwareID)
	if err != nil {
		return nil, err
	}
	return &fw, err
}

func (fs *firmwareService) UpdateResourceTags(id model.UserKey, collectionID, identifier string, res interface{}) error {
	fw := res.(*model.Firmware)
	return fs.store.UpdateFirmwareTags(id, identifier, fw.Tags)
}

func (fs *firmwareService) ListFirmwareTags(ctx context.Context, req *apipb.TagRequest) (*apipb.TagResponse, error) {
	return listTags(ctx, req, fs)
}

func (fs *firmwareService) UpdateFirmwareTags(ctx context.Context, req *apipb.UpdateTagRequest) (*apipb.TagResponse, error) {
	return updateTags(ctx, req, fs)
}

func (fs *firmwareService) GetFirmwareTag(ctx context.Context, req *apipb.TagRequest) (*apipb.TagValueResponse, error) {
	return getTag(ctx, req, fs)
}

func (fs *firmwareService) DeleteFirmwareTag(ctx context.Context, req *apipb.TagRequest) (*apipb.TagValueResponse, error) {
	return deleteTag(ctx, req, fs)
}

func (fs *firmwareService) UpdateFirmwareTag(ctx context.Context, req *apipb.TagRequest) (*apipb.TagValueResponse, error) {
	return updateTag(ctx, req, fs)
}
