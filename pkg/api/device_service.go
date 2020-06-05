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
	"strconv"

	"github.com/ExploratoryEngineering/logging"
	"github.com/eesrc/horde/pkg/addons/magpie/datastore"
	"github.com/eesrc/horde/pkg/api/apipb"
	"github.com/eesrc/horde/pkg/api/apitoolbox"
	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// newDeviceService creates a new implementation of the apipb.DevicesServer
// server.
func newDeviceService(
	store storage.DataStore,
	dataStoreClient datastore.DataStoreClient,
	sender DownstreamMessageSender) deviceService {
	return deviceService{
		store:           store,
		dataStoreClient: dataStoreClient,
		defaultGrpcAuth: defaultGrpcAuth{Store: store},
		sender:          sender,
	}
}

type deviceService struct {
	store           storage.DataStore
	dataStoreClient datastore.DataStoreClient
	sender          DownstreamMessageSender

	defaultGrpcAuth
}

func (d *deviceService) loadCollection(auth *authResult, collectionID string) (model.Collection, error) {
	collID, err := model.NewCollectionKeyFromString(collectionID)
	if err != nil {
		return model.Collection{}, status.Error(codes.InvalidArgument, "Invalid collection ID")
	}
	collection, err := d.store.RetrieveCollection(auth.User.ID, collID)
	if err != nil {
		if err == storage.ErrNotFound {
			return model.Collection{}, status.Error(codes.NotFound, "Unknown collection")
		}
		logging.Warning("Unable to read collection %d: %v", collID, err)
		return model.Collection{}, status.Error(codes.Internal, "Unable to read collection")
	}
	return collection, nil
}

func (d *deviceService) CreateDevice(ctx context.Context, req *apipb.Device) (*apipb.Device, error) {
	if req == nil || req.CollectionId == nil {
		return nil, status.Error(codes.InvalidArgument, "Missing collection/device ID")
	}
	auth, err := d.EnsureAuth(ctx)
	if err != nil {
		return nil, err
	}

	device := model.NewDevice()
	if req.Imsi == nil || req.Imei == nil {
		return nil, status.Error(codes.InvalidArgument, "Missing IMSI/IMEI on device")
	}

	imsiV, err := strconv.ParseInt(req.Imsi.Value, 10, 63)
	if err != nil || imsiV <= 0 {
		return nil, status.Error(codes.InvalidArgument, "Invalid IMSI on device")
	}
	imeiV, err := strconv.ParseInt(req.Imei.Value, 10, 63)
	if err != nil || imeiV <= 0 {
		return nil, status.Error(codes.InvalidArgument, "Invalid IMEI on device")
	}

	device.IMSI = imsiV
	device.IMEI = imeiV

	if req.Tags != nil {
		for k, v := range req.Tags {
			if !device.IsValidTag(k, v) {
				return nil, status.Error(codes.InvalidArgument, "Invalid tag name/value")
			}
			device.SetTag(k, v)
		}
	}

	coll, err := d.loadCollection(auth, req.CollectionId.Value)
	if err != nil {
		return nil, err
	}

	device.CollectionID = coll.ID
	device.ID = d.store.NewDeviceID()

	if coll.Firmware.Management == model.CollectionManagement {
		device.Firmware.TargetFirmwareID = coll.Firmware.TargetFirmwareID
	}

	if err := d.store.CreateDevice(auth.User.ID, device); err != nil {
		if err == storage.ErrAccess {
			return nil, status.Error(codes.PermissionDenied, "Must be administrator to create device")
		}
		if err == storage.ErrAlreadyExists {
			return nil, status.Error(codes.AlreadyExists, "IMSI/IMEI is already in use by another device")
		}
		logging.Warning("Unable to create device on collection %d: %v", coll.ID, err)
		return nil, status.Error(codes.Internal, "Unable to create device")
	}

	return apitoolbox.NewDeviceFromModel(device, coll), nil
}

func (d *deviceService) RetrieveDevice(ctx context.Context, req *apipb.DeviceRequest) (*apipb.Device, error) {
	if req == nil || req.CollectionId == nil || req.DeviceId == nil {
		return nil, status.Error(codes.InvalidArgument, "Missing collection/device ID")
	}
	auth, err := d.EnsureAuth(ctx)
	if err != nil {
		return nil, err
	}

	device, err := d.loadDevice(auth, req.CollectionId.Value, req.DeviceId.Value)
	if err != nil {
		return nil, err
	}
	coll, err := d.loadCollection(auth, req.CollectionId.Value)
	if err != nil {
		return nil, err
	}
	return apitoolbox.NewDeviceFromModel(device, coll), nil
}

func (d *deviceService) UpdateDevice(ctx context.Context, req *apipb.UpdateDeviceRequest) (*apipb.Device, error) {
	if req == nil || req.ExistingCollectionId == nil || req.DeviceId == nil {
		return nil, status.Error(codes.InvalidArgument, "Missing collection/device ID")
	}
	auth, err := d.EnsureAuth(ctx)
	if err != nil {
		return nil, err
	}

	device, err := d.loadDevice(auth, req.ExistingCollectionId.Value, req.DeviceId.Value)
	if err != nil {
		return nil, err
	}
	coll, err := d.loadCollection(auth, req.ExistingCollectionId.Value)
	if err != nil {
		return nil, err
	}

	if req.Firmware != nil && coll.Firmware.Management == model.CollectionManagement {
		return nil, status.Error(codes.InvalidArgument, "Firmware is managed by the collection")
	}

	update := false
	if req.Imei != nil {
		imeiV, err := strconv.ParseInt(req.Imei.Value, 10, 63)
		if err != nil || imeiV <= 0 {
			return nil, status.Error(codes.InvalidArgument, "Invalid IMEI")
		}

		if imeiV != device.IMEI {
			device.IMEI = imeiV
			update = true
		}
	}
	if req.Imsi != nil {
		imsiV, err := strconv.ParseInt(req.Imsi.Value, 10, 63)
		if err != nil || imsiV <= 0 {
			return nil, status.Error(codes.InvalidArgument, "Invalid IMSI")
		}
		if imsiV != device.IMSI {
			device.IMSI = imsiV
			update = true
		}
	}

	if req.Tags != nil {
		for k, v := range req.Tags {
			if !device.IsValidTag(k, v) {
				return nil, status.Error(codes.InvalidArgument, "Invalid tag name/value combination")
			}
			device.SetTag(k, v)
			update = true
		}
	}

	if req.CollectionId != nil {
		newCollID, err := model.NewCollectionKeyFromString(req.CollectionId.Value)
		logging.Error("Got new collection ID: %s :%v", req.CollectionId.Value, err)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "Invalid collection ID")
		}
		if newCollID != coll.ID {
			// update collection ID
			device.CollectionID = newCollID
			// reset firmware state if the collection changes
			device.Firmware.CurrentFirmwareID = 0
			device.Firmware.TargetFirmwareID = 0
			device.Firmware.State = model.Current
			device.Firmware.StateMessage = ""
			update = true
		}
	}

	if req.Firmware != nil {
		checkFW := false
		if req.Firmware.CurrentFirmwareId != nil {
			// update current firmware id
			newID, err := model.NewFirmwareKeyFromString(req.Firmware.CurrentFirmwareId.Value)
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, "Invalid current firmware ID")
			}
			device.Firmware.CurrentFirmwareID = newID
			update = true
			checkFW = true
		}
		if req.Firmware.TargetFirmwareId != nil {
			// update target firmware ID
			newID, err := model.NewFirmwareKeyFromString(req.Firmware.TargetFirmwareId.Value)
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, "Invalid target firmware ID")
			}
			device.Firmware.TargetFirmwareID = newID
			update = true
			checkFW = true
		}
		if checkFW {
			device.Firmware.State = model.Pending
			if device.Firmware.TargetFirmwareID == device.Firmware.CurrentFirmwareID || device.Firmware.TargetFirmwareID == 0 {
				device.Firmware.State = model.Current
			}
			if _, _, err := d.store.RetrieveCurrentAndTargetFirmware(device.CollectionID, device.Firmware.CurrentFirmwareID, device.Firmware.TargetFirmwareID); err != nil {
				if err == storage.ErrNotFound {
					return nil, status.Error(codes.NotFound, "Unknown firmware ID")
				}
				logging.Warning("Unable to read firmware ID (current=%d, target=%d) for collection %d: %v",
					device.Firmware.CurrentFirmwareID, device.Firmware.TargetFirmwareID, device.CollectionID, err)
				return nil, status.Error(codes.Internal, "Unable to check firmware")
			}
		}
	}

	if update {
		if err := d.store.UpdateDevice(auth.User.ID, coll.ID, device); err != nil {
			if err == storage.ErrAccess {
				return nil, status.Error(codes.PermissionDenied, "Must be administrator to update device")
			}
			if err == storage.ErrNotFound {
				// Collection or device is owned by someone else
				return nil, status.Error(codes.NotFound, "Unknown device or collection")
			}
			logging.Warning("Error updating device %d (collection ID=%d): %v", device.ID, coll.ID, err)
			return nil, status.Error(codes.Internal, "Unable to update device")
		}
	}
	// update the device
	return apitoolbox.NewDeviceFromModel(device, coll), nil
}

func (d *deviceService) DeleteDevice(ctx context.Context, req *apipb.DeviceRequest) (*apipb.Device, error) {
	if req == nil || req.CollectionId == nil || req.DeviceId == nil {
		return nil, status.Error(codes.InvalidArgument, "Missing collection/device ID")
	}
	auth, err := d.EnsureAuth(ctx)
	if err != nil {
		return nil, err
	}

	device, err := d.loadDevice(auth, req.CollectionId.Value, req.DeviceId.Value)
	if err != nil {
		return nil, err
	}
	coll, err := d.loadCollection(auth, req.CollectionId.Value)
	if err != nil {
		return nil, err
	}
	if err := d.store.DeleteDevice(auth.User.ID, coll.ID, device.ID); err != nil {
		if err == storage.ErrAccess {
			return nil, status.Error(codes.PermissionDenied, "Must be administrator to delete device")
		}
		logging.Warning("Unable to remove device %d (collection ID = %d): %v", device.ID, coll.ID, err)
		return nil, status.Error(codes.Internal, "Unable to remove device")
	}
	return apitoolbox.NewDeviceFromModel(device, coll), nil
}

func (d *deviceService) ClearFirmwareError(ctx context.Context, req *apipb.DeviceRequest) (*apipb.ClearFirmwareErrorResponse, error) {
	if req == nil || req.CollectionId == nil || req.DeviceId == nil {
		return nil, status.Error(codes.InvalidArgument, "Missing collection/device ID")
	}
	auth, err := d.EnsureAuth(ctx)
	if err != nil {
		return nil, err
	}

	device, err := d.loadDevice(auth, req.CollectionId.Value, req.DeviceId.Value)
	if err != nil {
		return nil, err
	}
	device.Firmware.State = model.Pending
	device.Firmware.StateMessage = ""
	if err := d.store.UpdateDevice(auth.User.ID, device.CollectionID, device); err != nil {
		return nil, status.Error(codes.Internal, "Unable to update state on device")
	}
	return &apipb.ClearFirmwareErrorResponse{}, nil
}

func (d *deviceService) ListDevices(ctx context.Context, req *apipb.ListDevicesRequest) (*apipb.ListDevicesResponse, error) {
	if req == nil || req.CollectionId == nil {
		return nil, status.Error(codes.InvalidArgument, "Missing collection ID")
	}
	auth, err := d.EnsureAuth(ctx)
	if err != nil {
		return nil, err
	}
	collection, err := d.loadCollection(auth, req.CollectionId.Value)
	if err != nil {
		return nil, err
	}
	devices, err := d.store.ListDevices(auth.User.ID, collection.ID)
	if err != nil {
		logging.Warning("Unable to read device list for collection %d: %v", collection.ID, err)
		return nil, status.Error(codes.Internal, "Unable to read device list")
	}
	ret := &apipb.ListDevicesResponse{
		Devices: make([]*apipb.Device, 0),
	}
	for _, v := range devices {
		ret.Devices = append(ret.Devices, apitoolbox.NewDeviceFromModel(v, collection))
	}
	return ret, nil
}

func (d *deviceService) ListDeviceMessages(ctx context.Context, req *apipb.ListMessagesRequest) (*apipb.ListMessagesResponse, error) {
	if req == nil || req.CollectionId == nil || req.DeviceId == nil {
		return nil, status.Error(codes.InvalidArgument, "Missing collection/device ID")
	}
	auth, err := d.EnsureAuth(ctx)
	if err != nil {
		return nil, err
	}

	device, err := d.loadDevice(auth, req.CollectionId.Value, req.DeviceId.Value)
	if err != nil {
		return nil, err
	}
	collection, err := d.loadCollection(auth, req.CollectionId.Value)
	if err != nil {
		return nil, err
	}

	dataFilter := &datastore.DataFilter{
		CollectionId: req.CollectionId.Value,
		DeviceId:     req.DeviceId.Value,
	}
	apitoolbox.ApplyDataFilter(req, dataFilter)

	result, err := d.dataStoreClient.GetData(ctx, dataFilter)
	if err != nil {
		logging.Warning("Error retrieving data from data store for device %d (collection ID=%d): %v", device.ID, device.CollectionID, err)
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
			logging.Warning("Error unmarshaling metadata: %v", err)
			continue
		}
		ret.Messages = append(ret.Messages, dataMessage)
	}
	return ret, nil
}

func (d *deviceService) SendMessage(ctx context.Context, req *apipb.SendMessageRequest) (*apipb.SendMessageResponse, error) {
	if req == nil || req.CollectionId == nil || req.DeviceId == nil {
		return nil, status.Error(codes.InvalidArgument, "Missing collection ID")
	}
	auth, err := d.EnsureAuth(ctx)
	if err != nil {
		return nil, err
	}

	device, err := d.loadDevice(auth, req.CollectionId.Value, req.DeviceId.Value)
	if err != nil {
		return nil, err
	}

	msg, err := apitoolbox.NewDownstreamMessage(req)
	if err != nil {
		return nil, err
	}
	if err := d.sender.Send(device, msg); err != nil {
		// There are multiple alternatives here. We're returning 409 conflict
		// which *technically* isn't correct but the device is in a state that
		// we have no control over so it's the closes. Another alternative
		// would be 503 unavailable which indicates a transient error but it
		// might not be transient if the device isn't connected.
		return nil, status.Error(codes.AlreadyExists, err.Error())
	}
	return &apipb.SendMessageResponse{}, nil
}

// Tag implementation. This is going to be a bit different since it uses both
// a collection ID and a device ID to retrieve the device as opposed to the
// team/collection/token tag updates that uses a single identifier for the
// resource. The net result is the same since the
func (d *deviceService) LoadTaggedResource(auth *authResult, collectionID string, identifier string) (taggedResource, error) {
	dev, err := d.loadDevice(auth, collectionID, identifier)
	if err != nil {
		return nil, err
	}
	return &dev, err
}

func (d *deviceService) loadDevice(auth *authResult, collID, identifier string) (model.Device, error) {
	deviceID, err := model.NewDeviceKeyFromString(identifier)
	if err != nil {
		return model.Device{}, status.Error(codes.InvalidArgument, "Invalid device ID")
	}
	collectionID, err := model.NewCollectionKeyFromString(collID)
	if err != nil {
		return model.Device{}, status.Error(codes.InvalidArgument, "Invalid collection ID")
	}
	device, err := d.store.RetrieveDevice(auth.User.ID, collectionID, deviceID)
	if err != nil {
		if err == storage.ErrNotFound {
			return model.Device{}, status.Error(codes.NotFound, "Unknown device")
		}
		logging.Warning("Error retrieving device %d (collectionID=%d): %v", deviceID, collectionID, err)
		return model.Device{}, status.Error(codes.Internal, "Unable to retrieve device")
	}
	return device, nil
}

func (d *deviceService) UpdateResourceTags(id model.UserKey, collectionID, identifier string, res interface{}) error {
	dev := res.(*model.Device)
	return d.store.UpdateDeviceTags(id, identifier, dev.Tags)
}

func (d *deviceService) ListDeviceTags(ctx context.Context, req *apipb.TagRequest) (*apipb.TagResponse, error) {
	return listTags(ctx, req, d)
}

func (d *deviceService) UpdateDeviceTags(ctx context.Context, req *apipb.UpdateTagRequest) (*apipb.TagResponse, error) {
	return updateTags(ctx, req, d)
}

func (d *deviceService) GetDeviceTag(ctx context.Context, req *apipb.TagRequest) (*apipb.TagValueResponse, error) {
	return getTag(ctx, req, d)
}

func (d *deviceService) DeleteDeviceTag(ctx context.Context, req *apipb.TagRequest) (*apipb.TagValueResponse, error) {
	return deleteTag(ctx, req, d)
}

func (d *deviceService) UpdateDeviceTag(ctx context.Context, req *apipb.TagRequest) (*apipb.TagValueResponse, error) {
	return updateTag(ctx, req, d)
}
