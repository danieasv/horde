package ctrlh

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
	"fmt"
	"time"

	"github.com/eesrc/horde/pkg/managementproto"
	"github.com/eesrc/horde/pkg/utils/grpcutil"
)

const grpcServerTimeout = 10 * time.Second

// gRPC management server setup
func connectToManagementServer(param ManagementServerParameters) managementproto.HordeManagementServiceClient {

	clientparam := grpcutil.GRPCClientParam{
		ServerEndpoint:     param.Endpoint,
		TLS:                param.TLS,
		CAFile:             param.CertFile,
		ServerHostOverride: param.HostnameOverride,
	}

	conn, err := grpcutil.NewGRPCClientConnection(clientparam)
	if err != nil {
		fmt.Printf("Unable to connect to management server: %v\n", err)
		return nil
	}
	return managementproto.NewHordeManagementServiceClient(conn)
}
