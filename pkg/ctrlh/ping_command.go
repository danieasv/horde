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
	"context"
	"fmt"

	"github.com/eesrc/horde/pkg/utils/grpcutil"
	"google.golang.org/grpc/connectivity"
)

// PingCommand is a subcommand that checks if the management service is
// reachable.
type PingCommand struct {
}

// Run connects to the management service and ensures that the connection
// changes state to Ready.
func (c *PingCommand) Run(rc RunContext) error {

	clientparam := grpcutil.GRPCClientParam{
		ServerEndpoint:     rc.HordeServer().Endpoint,
		TLS:                rc.HordeServer().TLS,
		CAFile:             rc.HordeServer().CertFile,
		ServerHostOverride: rc.HordeServer().HostnameOverride,
	}

	conn, err := grpcutil.NewGRPCClientConnection(clientparam)
	if err != nil {
		fmt.Printf("Unable to connect to management server: %v\n", err)
		return errStd
	}

	ctx, done := context.WithTimeout(context.Background(), grpcServerTimeout)
	defer done()
	if !conn.WaitForStateChange(ctx, connectivity.Ready) {
		fmt.Printf("Connection did not change state to ready")
		return errStd
	}

	fmt.Println("OK")
	return nil
}
