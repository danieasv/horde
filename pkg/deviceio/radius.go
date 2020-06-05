package deviceio

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
	"github.com/eesrc/horde/pkg/apn/radius"

	"github.com/eesrc/horde/pkg/deviceio/rxtx"
)

// RADIUSServer is a gRPC-backed RADIUS server
type RADIUSServer struct {
	config radius.ServerParameters
	server radius.Server
	client rxtx.RADIUSClient
}

// NewRADIUSServer creates a new gRPC-backed RADIUS server
func NewRADIUSServer(client rxtx.RADIUSClient, config radius.ServerParameters) *RADIUSServer {
	ret := &RADIUSServer{
		config: config,
		client: client,
	}
	ret.server = radius.NewRADIUSServer(config, ret.accessHandler)
	return ret
}

// Start launches the RADIUS service
func (r *RADIUSServer) Start() error {
	return r.server.Start()
}

// Stop stops the RADIUS service
func (r *RADIUSServer) Stop() {
	r.server.Stop()
}

func (r *RADIUSServer) accessHandler(req radius.AccessRequest) radius.AccessResponse {
	imsi, err := strconv.ParseInt(req.IMSI, 10, 63)
	if err != nil {
		logging.Info("Rejecting invalid IMSI from NAS %s: %s", req.NASIdentifier, req.IMSI)
		return radius.AccessResponse{
			Accept:        false,
			RejectMessage: "Missing or invalid IMSI",
		}
	}
	ar := &rxtx.AccessRequest{
		NasIdentifier:    req.NASIdentifier,
		Imsi:             imsi,
		Username:         req.Username,
		UserLocationInfo: req.UserLocationInfo,
		Imeisv:           req.IMEISV,
		ImsiMccMnc:       req.IMSIMccMnc,
		MsTimezone:       req.MSTimezone,
		NasIpAddress:     req.NASIPAddress,
		Password:         []byte(req.Password),
	}
	ctx, done := context.WithTimeout(context.Background(), grpcTimeout)
	defer done()

	resp, err := r.client.Access(ctx, ar)
	if err != nil {
		// Make a 2nd try
		ctx2, done2 := context.WithTimeout(context.Background(), grpcRetryTimeout)
		defer done2()
		resp, err = r.client.Access(ctx2, ar)
		if err != nil {
			logging.Error("Unable to query for access-request (IMIS=%s, NAS=%s): %v", req.IMSI, req.NASIdentifier, err)
			return radius.AccessResponse{
				Accept:        false,
				RejectMessage: "Service unavailable",
			}
		}
	}
	return radius.AccessResponse{
		Accept:        resp.Accepted,
		IPAddress:     resp.IpAddress,
		RejectMessage: resp.Message,
	}
}
