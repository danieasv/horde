package main

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
	"fmt"
	"os"

	"github.com/eesrc/horde/pkg/storage/sqlstore"
	"github.com/eesrc/horde/pkg/utils/grpcutil"
	"github.com/eesrc/horde/pkg/version"

	"github.com/eesrc/horde/pkg/metrics"

	"github.com/eesrc/horde/pkg/utils"

	"github.com/ExploratoryEngineering/logging"
	"github.com/ExploratoryEngineering/params"
	"github.com/eesrc/horde/pkg/addons/magpie"
)

type args struct {
	Log                utils.LogParameters
	MonitoringEndpoint string `param:"desc=Monitoring endpoint;default=localhost:0"`
	GRPC               grpcutil.GRPCServerParam
	SQL                sqlstore.Parameters
	Version            bool `param:"desc=Show version, then exit;default=false"`
}

func main() {
	var cfg args
	if err := params.NewEnvFlag(&cfg, os.Args[1:]); err != nil {
		fmt.Println(err.Error())
		return
	}
	if cfg.Version {
		fmt.Println(version.Release())
		return
	}
	utils.InitLogs("magpie", cfg.Log)

	logging.Info("Magpie service is launching. Version is %s (%s)", version.Number, version.Name)
	server, err := magpie.NewDataServer(cfg.SQL)
	if err != nil {
		logging.Error("Unable to create server: %v", err)
		return
	}

	monitoring, err := metrics.NewMonitoringServer(cfg.MonitoringEndpoint)
	if err != nil {
		logging.Error("Unable to create metrics endpoint: %v", err)
		return
	}
	if err := monitoring.Start(); err != nil {
		logging.Error("Unable to start metrics endpoint: %v", err)
		return
	}
	logging.Info("Metrics endpoint is at %s", monitoring.ServerURL())

	list, err := magpie.StartServer(server, cfg.GRPC)
	if err != nil {
		logging.Error("Unable to launch gRPC service: %v", err)
		return
	}
	logging.Info("Data server listening on %s", list)

	utils.WaitForSignal()
}
