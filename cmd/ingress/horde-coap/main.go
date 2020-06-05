package main

import (
	"fmt"
	"os"

	"github.com/eesrc/horde/pkg/deviceio"

	"github.com/ExploratoryEngineering/logging"
	"github.com/ExploratoryEngineering/params"
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

type cmdParam struct {
	GRPC grpcutil.GRPCClientParam
	COAP deviceio.CoAPParameters
	Log  utils.LogParameters
}

func main() {
	var config cmdParam

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

	ut := deviceio.NewCoAPServer(rxtx.NewRxtxClient(conn), config.COAP)
	if err := ut.Start(); err != nil {
		logging.Error("Unable to start the CoAP listener: %v", err)
		os.Exit(3)
	}
	defer ut.Stop()

	logging.Info("%s started, version=%s (%s)", os.Args[0], version.Number, version.Name)
	utils.WaitForSignal()

}
