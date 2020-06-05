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
	"github.com/eesrc/horde/pkg/apn/radius"
	"github.com/eesrc/horde/pkg/deviceio"
	"github.com/eesrc/horde/pkg/fota"
	"github.com/eesrc/horde/pkg/ghlogin"
	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage/sqlstore"
	"github.com/eesrc/horde/pkg/utils"
	"github.com/eesrc/horde/pkg/utils/grpcutil"

	"github.com/eesrc/horde/pkg/restapi"
)

type dataStoreParams struct {
	GRPC grpcutil.GRPCServerParam
	SQL  sqlstore.Parameters
}

// hordeParameters holds the service parameters.
type hordeParameters struct {
	DataCenterID       int `param:"desc=Data center ID (1-255);default=1;min=1;max=255"`
	WorkerID           int `param:"desc=Worker ID;default=1;min=0;max=65535"`
	Log                utils.LogParameters
	DB                 sqlstore.Parameters
	ExitAfterCreate    bool `param:"desc=Exit after schema has been created;default=false"`
	ShowSchema         bool `param:"desc=Print database schema to stdout;default=false"`
	Caching            bool `param:"desc=Use memory cache for data store;default=false"`
	HTTP               restapi.ServerParameters
	Connect            restapi.ConnectIDParameters
	Github             ghlogin.Config
	GRPCDataStore      grpcutil.GRPCClientParam
	LaunchDataStorage  bool   `param:"desc=Launch embedded data storage server;default=false"`
	MonitoringEndpoint string `param:"desc=Monitoring (varz) and trace endpoint;default=127.0.0.1:0"`
	EnableLocalOutputs bool   `param:"desc=Enable outputs to local IP range;default=false"`
	DataStorage        dataStoreParams
	DeviceFieldMask    model.FieldMaskParameters
	Management         grpcutil.GRPCServerParam
	AuditLog           bool `param:"desc=Device audit logging;default=true"`
	RADIUS             radius.ServerParameters
	RADIUSGrpc         grpcutil.GRPCServerParam
	EmbeddedRADIUS     bool `param:"desc=Launch embedded RADIUS server;default=true"`
	RxTxGRPC           grpcutil.GRPCServerParam
	EmbeddedListener   bool `param:"desc=Launch embedded listeners (UDP/CoAP);default=true"`
	EmbeddedCOAP       deviceio.CoAPParameters
	EmbeddedUDP        deviceio.UDPParameters
	FOTA               fota.Parameters
	Version            bool `param:"desc=Show version;default=false"`
}
