package server

//
//Copyright 2019 Telenor Digital AS
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.
//
import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/ExploratoryEngineering/logging"
	"github.com/eesrc/horde/pkg/managementproto"
	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
	"github.com/eesrc/horde/pkg/utils/grpcutil"
	"google.golang.org/grpc"
)

// This is the management service implementation. This is more or less a quite
// thin wrapper around the storage interface. The APN management functions
// aren't something that will be used frequently (except for diagnostics and
// then it's just querying the allocations)

const (
	maxAllocationsReturned = 1000
)

// StartHordeManagementInterface starts the gRPC management interface. If one
// of the store parameters is nil the store won't support operations on that
// particular store.
func StartHordeManagementInterface(config grpcutil.GRPCServerParam, apnStore storage.APNStore, mainStore storage.DataStore, apnConfig *storage.APNConfigCache) error {
	server, err := grpcutil.NewGRPCServer(config)
	if err != nil {
		return err
	}
	mgmtServer := newManagementServer(apnStore, mainStore, apnConfig)

	if err := server.Launch(func(srv *grpc.Server) {
		managementproto.RegisterHordeManagementServiceServer(srv, mgmtServer)
	}, 200*time.Millisecond); err != nil {
		logging.Error("Error launching Horde management service: %v", err)
		return err
	}
	return nil
}

type hordeManagementServer struct {
	apnStore  storage.APNStore
	mainStore storage.DataStore
	apnCache  *storage.APNConfigCache
}

// newManagementServer creates a new management server. If the mainStore is omitted
// it will only support APN operations.
func newManagementServer(apnStore storage.APNStore, mainStore storage.DataStore, apnConfig *storage.APNConfigCache) managementproto.HordeManagementServiceServer {
	return &hordeManagementServer{
		apnStore:  apnStore,
		mainStore: mainStore,
		apnCache:  apnConfig,
	}
}

func makeResult(success bool, message string) *managementproto.Result {
	return &managementproto.Result{
		Success: success,
		Error:   message,
	}
}

func (m *hordeManagementServer) AddAPN(ctx context.Context, req *managementproto.AddAPNRequest) (*managementproto.AddAPNResponse, error) {
	if m.apnStore == nil {
		return nil, errors.New("apn operations is not supported by this management server")
	}
	if req.NewAPN.ApnID == 0 {
		return &managementproto.AddAPNResponse{Result: makeResult(false, "APN ID cannot be 0")}, nil
	}
	if req.NewAPN.Name == "" {
		return &managementproto.AddAPNResponse{Result: makeResult(false, "APN name can not be empty")}, nil
	}
	newAPN := model.APN{
		ID:   int(req.NewAPN.ApnID),
		Name: req.NewAPN.Name,
	}
	if err := m.apnStore.CreateAPN(newAPN); err != nil {
		return &managementproto.AddAPNResponse{Result: makeResult(false, err.Error())}, nil
	}

	return &managementproto.AddAPNResponse{Result: makeResult(true, "")}, nil
}

func (m *hordeManagementServer) RemoveAPN(ctx context.Context, req *managementproto.RemoveAPNRequest) (*managementproto.RemoveAPNResponse, error) {
	if m.apnStore == nil {
		return nil, errors.New("apn operations is not supported by this management server")
	}
	if err := m.apnStore.RemoveAPN(int(req.ApnID)); err != nil {
		return &managementproto.RemoveAPNResponse{Result: makeResult(false, err.Error())}, nil
	}
	return &managementproto.RemoveAPNResponse{Result: makeResult(true, "")}, nil
}

func (m *hordeManagementServer) AddNAS(ctx context.Context, req *managementproto.AddNASRequest) (*managementproto.AddNASResponse, error) {
	if m.apnStore == nil {
		return nil, errors.New("apn operations is not supported by this management server")
	}
	if req.ApnID == 0 {
		return &managementproto.AddNASResponse{Result: makeResult(false, "APN ID must be set")}, nil
	}
	if req.NewRange.NasID == 0 {
		return &managementproto.AddNASResponse{Result: makeResult(false, "NAS ID must be set")}, nil
	}
	if req.NewRange.NasIdentifier == "" {
		return &managementproto.AddNASResponse{Result: makeResult(false, "NAS identifier cannot be blank")}, nil
	}
	if req.NewRange.CIDR == "" {
		return &managementproto.AddNASResponse{Result: makeResult(false, "CIDR cannot be blank")}, nil
	}
	_, _, err := net.ParseCIDR(req.NewRange.CIDR)
	if err != nil {
		return &managementproto.AddNASResponse{Result: makeResult(false, "Invalid CIDR range")}, nil
	}
	newNAS := model.NAS{
		ID:         int(req.NewRange.NasID),
		Identifier: req.NewRange.NasIdentifier,
		ApnID:      int(req.ApnID),
		CIDR:       req.NewRange.CIDR,
	}
	if err := m.apnStore.CreateNAS(newNAS); err != nil {
		return &managementproto.AddNASResponse{Result: makeResult(false, err.Error())}, nil
	}
	return &managementproto.AddNASResponse{Result: makeResult(true, "")}, nil
}

func (m *hordeManagementServer) AddAllocation(ctx context.Context, req *managementproto.AddAllocationRequest) (*managementproto.AddAllocationResponse, error) {
	if m.apnStore == nil {
		return nil, errors.New("apn operations is not supported by this management server")
	}
	if req.ApnID < 0 {
		return &managementproto.AddAllocationResponse{Result: makeResult(false, "APN ID must be set")}, nil
	}
	if req.NasID < 0 {
		return &managementproto.AddAllocationResponse{Result: makeResult(false, "NAS ID must be set")}, nil
	}
	if req.IP == "" {
		return &managementproto.AddAllocationResponse{Result: makeResult(false, "IP must be specified")}, nil
	}
	if req.IMSI == 0 {
		return &managementproto.AddAllocationResponse{Result: makeResult(false, "IMSI must be set")}, nil
	}

	newAlloc := model.Allocation{
		IP:      net.ParseIP(req.IP),
		IMSI:    req.IMSI,
		ApnID:   int(req.ApnID),
		NasID:   int(req.NasID),
		Created: time.Now(),
	}

	if err := m.apnStore.CreateAllocation(newAlloc); err != nil {
		return &managementproto.AddAllocationResponse{Result: makeResult(false, err.Error())}, nil
	}
	return &managementproto.AddAllocationResponse{Result: makeResult(true, "")}, nil
}

func (m *hordeManagementServer) RemoveNAS(ctx context.Context, req *managementproto.RemoveNASRequest) (*managementproto.RemoveNASResponse, error) {
	if m.apnStore == nil {
		return nil, errors.New("apn operations is not supported by this management server")
	}
	if err := m.apnStore.RemoveNAS(int(req.ApnID), int(req.NasID)); err != nil {
		return &managementproto.RemoveNASResponse{Result: makeResult(false, err.Error())}, nil
	}
	return &managementproto.RemoveNASResponse{Result: makeResult(true, "")}, nil
}

func (m *hordeManagementServer) ListAPNAllocations(ctx context.Context, req *managementproto.ListAPNAllocationsRequest) (*managementproto.ListAPNAllocationsResponse, error) {
	if m.apnStore == nil {
		return nil, errors.New("apn operations is not supported by this management server")
	}
	ret := &managementproto.ListAPNAllocationsResponse{}
	list, err := m.apnStore.ListAllocations(int(req.ApnID), int(req.NasID), maxAllocationsReturned)
	if err != nil {
		return &managementproto.ListAPNAllocationsResponse{
			Result: makeResult(false, err.Error()),
		}, nil
	}
	ret.Result = makeResult(true, "")
	for _, v := range list {
		allocation := &managementproto.APNAllocation{
			NasID:   int32(v.NasID),
			IP:      v.IP.String(),
			Created: v.Created.Unix(),
			IMSI:    v.IMSI,
			IMEI:    v.IMEI,
		}
		ret.Allocations = append(ret.Allocations, allocation)
	}
	return ret, nil
}

func (m *hordeManagementServer) RemoveAPNAllocation(ctx context.Context, req *managementproto.RemoveAPNAllocationRequest) (*managementproto.RemoveAPNAllocationResponse, error) {
	if m.apnStore == nil {
		return nil, errors.New("apn operations is not supported by this management server")
	}
	if err := m.apnStore.RemoveAllocation(int(req.ApnID), int(req.NasID), req.IMSI); err != nil {
		return &managementproto.RemoveAPNAllocationResponse{
			Result: makeResult(false, err.Error()),
		}, nil
	}

	return &managementproto.RemoveAPNAllocationResponse{Result: makeResult(true, "")}, nil
}

func (m *hordeManagementServer) ListAPN(ctx context.Context, req *managementproto.ListAPNRequest) (*managementproto.ListAPNResponse, error) {
	if m.apnStore == nil {
		return nil, errors.New("apn operations is not supported by this management server")
	}
	results := make([]*managementproto.APNConfig, 0)

	list, err := m.apnStore.ListAPN()
	if err != nil {
		return &managementproto.ListAPNResponse{Result: makeResult(false, err.Error())}, nil
	}
	for _, v := range list {
		apn := &managementproto.APNConfig{
			APN: &managementproto.APN{
				ApnID: int32(v.ID),
				Name:  v.Name,
			},
		}
		nasList, err := m.apnStore.ListNAS(v.ID)
		if err != nil {
			return &managementproto.ListAPNResponse{Result: makeResult(false, err.Error())}, nil
		}
		for _, nas := range nasList {
			apn.NasRanges = append(apn.NasRanges, &managementproto.NASRange{
				NasID:         int32(nas.ID),
				NasIdentifier: nas.Identifier,
				CIDR:          nas.CIDR,
			})
		}
		results = append(results, apn)
	}
	return &managementproto.ListAPNResponse{
		Result: makeResult(true, ""),
		APNs:   results,
	}, nil
}

func (m *hordeManagementServer) ReloadAPN(ctx context.Context, req *managementproto.ReloadAPNRequest) (*managementproto.ReloadAPNResponse, error) {
	if m.apnStore == nil {
		return &managementproto.ReloadAPNResponse{Result: makeResult(false, "This management server does not support APN operations")}, nil
	}
	if m.apnCache == nil {
		return &managementproto.ReloadAPNResponse{Result: makeResult(false, "This management server can't manage the APN cache")}, nil
	}
	if err := m.apnCache.Reload(m.apnStore); err != nil {
		return &managementproto.ReloadAPNResponse{Result: makeResult(false, err.Error())}, nil
	}

	return &managementproto.ReloadAPNResponse{
		Result: makeResult(true, ""),
	}, nil
}

func (m *hordeManagementServer) AddUser(ctx context.Context, req *managementproto.AddUserRequest) (*managementproto.AddUserResponse, error) {
	if m.mainStore == nil {
		return nil, errors.New("this management server does not support user operations")
	}
	t := model.NewTeam()
	t.ID = m.mainStore.NewTeamID()
	t.Tags.SetTag("name", "My private team")

	userID := m.mainStore.NewUserID()
	u := model.NewUser(userID, userID.String(), model.AuthInternal, t.ID)
	u.Name = req.Name
	u.Email = req.Email
	if req.Email != "" {
		u.VerifiedEmail = true
	}
	t.AddMember(model.NewMember(u, model.AdminRole))

	if err := m.mainStore.CreateUser(u, t); err != nil {
		logging.Warning("Unable to create internal user: %v", err)
		return &managementproto.AddUserResponse{
			Result: makeResult(false, "Unable to create user, consult application logs"),
		}, nil
	}

	token := model.NewToken()
	if err := token.GenerateToken(); err != nil {
		logging.Warning("Unable to generate token for user %s: %v", u.ID, err)
		return &managementproto.AddUserResponse{
			Result: makeResult(false, "Unable to generate token, consult application logs"),
		}, nil
	}
	token.UserID = u.ID
	token.Write = true
	token.Resource = "/"
	if err := m.mainStore.CreateToken(token); err != nil {
		logging.Warning("Unable to create API token for user %v: %v", u.ID, err)
		return &managementproto.AddUserResponse{
			Result: makeResult(false, "Unable to create API token, consult application logs"),
		}, nil
	}
	return &managementproto.AddUserResponse{
		Result:   makeResult(true, ""),
		UserId:   u.ID.String(),
		ApiToken: token.Token,
	}, nil
}

func (m *hordeManagementServer) AddToken(ctx context.Context, req *managementproto.AddTokenRequest) (*managementproto.AddTokenResponse, error) {
	if m.mainStore == nil {
		return nil, errors.New("this management server does not support user operations")
	}
	userID, err := model.NewUserKeyFromString(req.UserId)
	if err != nil {
		return &managementproto.AddTokenResponse{
			Result: makeResult(false, "Invalid token format"),
		}, nil
	}
	user, err := m.mainStore.RetrieveUser(userID)
	if err != nil {
		if err != storage.ErrNotFound {
			logging.Warning("Unable to retrieve user %d: %v", userID, err)
		}
		return &managementproto.AddTokenResponse{
			Result: makeResult(false, "Unable to retrieve user, consult application logs"),
		}, nil
	}
	if user.AuthType != model.AuthInternal {
		return &managementproto.AddTokenResponse{
			Result: makeResult(false, "Can't create API tokens for non-internal users"),
		}, nil
	}
	newToken := model.NewToken()
	if err := newToken.GenerateToken(); err != nil {
		logging.Warning("Unable to generate token for user %d: %v", userID, err)
		return &managementproto.AddTokenResponse{
			Result: makeResult(false, "Unable to generate token, consult application logs"),
		}, nil
	}
	newToken.UserID = userID
	newToken.Resource = "/"
	newToken.Write = true
	if err := m.mainStore.CreateToken(newToken); err != nil {
		logging.Warning("Unable to store token for user %d: %v", userID, err)
		return &managementproto.AddTokenResponse{
			Result: makeResult(false, "Unable to store token, consult application logs"),
		}, nil
	}
	return &managementproto.AddTokenResponse{
		Result:   makeResult(true, ""),
		ApiToken: newToken.Token,
	}, nil

}

func (m *hordeManagementServer) RemoveToken(ctx context.Context, req *managementproto.RemoveTokenRequest) (*managementproto.RemoveTokenResponse, error) {
	if m.mainStore == nil {
		return nil, errors.New("this management server does not support user operations")
	}

	userID, err := model.NewUserKeyFromString(req.UserId)
	if err != nil {
		return &managementproto.RemoveTokenResponse{
			Result: makeResult(false, "Invalid token format"),
		}, nil
	}
	user, err := m.mainStore.RetrieveUser(userID)
	if err != nil {
		if err != storage.ErrNotFound {
			logging.Warning("Unable to retrieve user %d: %v", userID, err)
		}
		return &managementproto.RemoveTokenResponse{
			Result: makeResult(false, "Unable to retrieve user, consult application logs"),
		}, nil
	}
	if user.AuthType != model.AuthInternal {
		return &managementproto.RemoveTokenResponse{
			Result: makeResult(false, "Can't manage API tokens for non-internal users"),
		}, nil
	}

	if err := m.mainStore.DeleteToken(userID, req.ApiToken); err != nil {
		if err != storage.ErrNotFound {
			logging.Warning("Unable to remove token %s: %v", req.ApiToken, err)
		}
		return &managementproto.RemoveTokenResponse{
			Result: makeResult(false, "Unable to remove token, consult application logs"),
		}, nil
	}
	return &managementproto.RemoveTokenResponse{
		Result: makeResult(true, ""),
	}, nil
}
