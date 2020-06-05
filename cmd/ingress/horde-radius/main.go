package main

import (
	"fmt"
	"os"

	"github.com/ExploratoryEngineering/logging"
	"github.com/ExploratoryEngineering/params"
	"github.com/eesrc/horde/pkg/apn/radius"
	"github.com/eesrc/horde/pkg/deviceio"
	"github.com/eesrc/horde/pkg/deviceio/rxtx"
	"github.com/eesrc/horde/pkg/utils"
	"github.com/eesrc/horde/pkg/utils/grpcutil"
	"github.com/eesrc/horde/pkg/version"
)

//
//Copyright 2020 Telenor Digital AS
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

func main() {
	var config = struct {
		RADIUS radius.ServerParameters
		GRPC   grpcutil.GRPCClientParam
		Log    utils.LogParameters
	}{}

	args := os.Args
	if err := params.NewEnvFlag(&config, args[1:]); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	utils.InitLogs(args[0], config.Log)

	conn, err := grpcutil.NewGRPCClientConnection(config.GRPC)
	if err != nil {
		logging.Error("Unable to create the rxtx client: %v", err)
		os.Exit(2)
	}

	rs := deviceio.NewRADIUSServer(rxtx.NewRADIUSClient(conn), config.RADIUS)
	if err := rs.Start(); err != nil {
		logging.Error("Unable to start the RADIUS server: %v", err)
		os.Exit(3)
	}
	defer rs.Stop()

	logging.Info("%s started, version=%s (%s)", os.Args[0], version.Number, version.Name)
	utils.WaitForSignal()
}
