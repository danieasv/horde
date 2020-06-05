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
	"encoding/hex"
	"net"
	"time"

	"github.com/eesrc/horde/pkg/deviceio"
	"github.com/eesrc/horde/pkg/utils/grpcutil"
	"google.golang.org/grpc"

	"github.com/ExploratoryEngineering/logging"
	"github.com/eesrc/horde/pkg/apn/allocator"
	"github.com/eesrc/horde/pkg/apn/radius"
	"github.com/eesrc/horde/pkg/deviceio/rxtx"
	"github.com/eesrc/horde/pkg/metrics"
	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
)

// This is the server part of the gRPC-backed RADIUS server.

// NewRxtxRADIUSServer creates the server for the gRPC-backed RADIUS
// listener-slash-server.
func NewRxtxRADIUSServer(apnConfig *storage.APNConfigCache, store storage.DataStore, allocator allocator.DeviceAddressAllocator) (rxtx.RADIUSServer, error) {
	metrics.DefaultRADIUSCounters.Start(apnConfig)
	return &rxtxRADIUS{
		store:     store,
		allocator: allocator,
		apnConfig: apnConfig,
	}, nil
}

type rxtxRADIUS struct {
	store     storage.DataStore
	allocator allocator.DeviceAddressAllocator
	apnConfig *storage.APNConfigCache
}

func (r *rxtxRADIUS) Access(ctx context.Context, req *rxtx.AccessRequest) (*rxtx.AccessResponse, error) {
	// Check if this a known NAS
	nas, ok := r.apnConfig.FindNAS(req.NasIdentifier)
	if !ok {
		metrics.DefaultRADIUSCounters.RejectRequest("unknown")
		logging.Debug("Unknown NAS in request: %s. Rejecting request.", req.NasIdentifier)
		return &rxtx.AccessResponse{
			Accepted:  false,
			Message:   "Unknown NAS",
			IpAddress: nil,
		}, nil
	}

	device, err := r.store.RetrieveDeviceByIMSI(req.Imsi)
	if err != nil {
		metrics.DefaultRADIUSCounters.RejectRequest(req.NasIdentifier)
		if err == storage.ErrNotFound {
			logging.Debug("Unknown device: %d. Rejecting request", req.Imsi)
			return &rxtx.AccessResponse{
				Accepted:  false,
				Message:   "Device does not exist",
				IpAddress: nil,
			}, nil
		}
		logging.Warning("Got error doing IMSI lookup for IMSI %d: %v", req.Imsi, err)
		return &rxtx.AccessResponse{
			Accepted:  false,
			Message:   "Device lookup error",
			IpAddress: nil,
		}, nil
	}
	// Got device -- allocate device
	ip, allocated, err := r.allocator.AllocateIP(req.Imsi, nas.ID)
	if err != nil {
		logging.Warning("Unable to allocate IP for device %v: %v", req.Imsi, err)
		metrics.DefaultRADIUSCounters.RejectRequest(req.NasIdentifier)
		return &rxtx.AccessResponse{
			Accepted: false,
			Message:  "Unable to allocate IP",
		}, nil
	}

	r.updateDeviceTags(device, nas, ip, req)

	if allocated {
		metrics.DefaultRADIUSCounters.IPAllocated(req.NasIdentifier)
	} else {
		metrics.DefaultRADIUSCounters.IPReused(req.NasIdentifier)
	}
	metrics.DefaultRADIUSCounters.AcceptRequest(req.NasIdentifier)
	logging.Debug("Device with IMSI %d has the IP address %s", req.Imsi, ip.String())
	return &rxtx.AccessResponse{
		Accepted:  true,
		IpAddress: ip,
	}, nil
}

// updateDeviceTags updates the metadata tags on the device. If the call fails
// it will continue execution
func (r *rxtxRADIUS) updateDeviceTags(device model.Device, nas model.NAS, ip net.IP, req *rxtx.AccessRequest) {
	device.SetTag("3GPP-User-Location-Info", hex.EncodeToString(req.UserLocationInfo))
	device.SetTag("3GPP-MS-TimeZone", hex.EncodeToString(req.MsTimezone))
	device.SetTag("RADIUS-IP-address", ip.String())
	device.SetTag("RADIUS-Allocated-At", time.Now().Format(time.RFC3339))
	device.Network.ApnID = nas.ApnID
	device.Network.NasID = nas.ID
	device.Network.AllocatedIP = ip.String()
	device.Network.AllocatedAt = time.Now()
	if err := r.store.UpdateDeviceMetadata(device); err != nil {
		logging.Warning("Error updating device metadata for device with IMSI %d: %v", device.IMSI, err)
	}
}

// StartRADIUSgRPC launches the gRPC endpoint for the RADIUS server. The RADIUS
// server must be launched separately
func StartRADIUSgRPC(params grpcutil.GRPCServerParam, datastore storage.DataStore, apnStore storage.APNStore, apnConfig *storage.APNConfigCache) (grpcutil.GRPCServer, error) {
	svr, err := grpcutil.NewGRPCServer(params)
	if err != nil {
		return nil, err
	}

	allocator, err := allocator.NewWriteThroughAllocator(apnConfig, apnStore)
	if err != nil {
		return nil, err
	}

	server, err := NewRxtxRADIUSServer(apnConfig, datastore, allocator)
	if err != nil {
		return nil, err
	}

	if err := svr.Launch(func(s *grpc.Server) {
		rxtx.RegisterRADIUSServer(s, server)
	}, 250*time.Millisecond); err != nil {
		return nil, err
	}

	return svr, nil
}

// StartLocalRADIUS launches the local gRPC endpoint *and* the RADIUS listener
// locally, ie the RADIUS server is embedded.  There's no corresponding Stop
// function but when we're shutting down a local server there's no need.
func StartLocalRADIUS(radiusConfig radius.ServerParameters, datastore storage.DataStore, apnStore storage.APNStore, apnConfig *storage.APNConfigCache) error {
	logging.Info("Launching embedded RADIUS server")
	serverParams := grpcutil.GRPCServerParam{Endpoint: "127.0.0.1:0"}
	server, err := StartRADIUSgRPC(serverParams, datastore, apnStore, apnConfig)
	if err != nil {
		return err
	}
	conn, err := grpcutil.NewGRPCClientConnection(grpcutil.GRPCClientParam{
		ServerEndpoint: server.Endpoint(),
	})
	if err != nil {
		return err
	}

	rs := deviceio.NewRADIUSServer(rxtx.NewRADIUSClient(conn), radiusConfig)
	if err := rs.Start(); err != nil {
		return err
	}
	return nil
}
