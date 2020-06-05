package server

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

	"github.com/eesrc/horde/pkg/deviceio"
	"github.com/eesrc/horde/pkg/deviceio/rxtx"

	"github.com/eesrc/horde/pkg/fota"
	"github.com/eesrc/horde/pkg/storage/fwimage"
	"github.com/eesrc/horde/pkg/storage/memdb"
	"github.com/eesrc/horde/pkg/utils/audit"
	"github.com/eesrc/horde/pkg/utils/grpcutil"
	"github.com/eesrc/horde/pkg/version"
	"google.golang.org/grpc"

	"github.com/eesrc/horde/pkg/addons/magpie"
	"github.com/eesrc/horde/pkg/apn"
	"github.com/eesrc/horde/pkg/metrics"
	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
	"github.com/eesrc/horde/pkg/utils"

	"github.com/ExploratoryEngineering/logging"
	"github.com/ExploratoryEngineering/params"
	"github.com/eesrc/horde/pkg/apn/radius"
	"github.com/eesrc/horde/pkg/output"
	"github.com/eesrc/horde/pkg/restapi"
	"github.com/eesrc/horde/pkg/storage/counters"
	"github.com/eesrc/horde/pkg/storage/sqlstore"
)

// LaunchHorde launches the entire Horde server using the supplied command line
// arguments. This function never returns.
func LaunchHorde(args []string) {
	config := hordeParameters{}
	if err := params.NewEnvFlag(&config, args); err != nil {
		fmt.Printf("%v\n", err)
		return
	}
	if config.Version {
		fmt.Println(version.Release())
		return
	}
	if config.ShowSchema {
		// Print the schema in a (semi) user friendly way
		schema := sqlstore.NewSchema(config.DB.Type, sqlstore.DBSchema+"\n"+sqlstore.DBAPNSchema+"\n"+sqlstore.DBDataStoreSchema)
		for _, v := range schema.DDL() {
			fmt.Println(v)
		}
		return
	}
	if config.AuditLog {
		audit.Enable()
	}

	utils.InitLogs("horde", config.Log)

	forced := model.FieldNames(config.DeviceFieldMask.ForcedFields())
	defaultFields := model.FieldNames(config.DeviceFieldMask.DefaultFields())
	logging.Info("Forced field mask: %v", forced)
	logging.Info("Default field mask: %v", defaultFields)
	if err := config.DeviceFieldMask.Valid(); err != nil {
		logging.Error("Invalid default and forced fields. The default must be as strict or stricter than the forced field mask")
		return
	}
	logging.Info("Horde service launching. Version is %s (%s)", version.Number, version.Name)
	metrics.DefaultCoreCounters.Start()

	var store storage.DataStore
	var err error
	store, err = sqlstore.NewSQLStore(config.DB.Type, config.DB.ConnectionString, config.DB.CreateSchema,
		uint8(config.DataCenterID), uint16(config.WorkerID))
	if err != nil {
		logging.Error("Unable to create data store: %v", err)
		return
	}
	if config.ExitAfterCreate && config.DB.CreateSchema {
		logging.Info("Schema created. Exiting.")
		return
	}
	if config.Caching {
		cachedStore, err := memdb.PrimeMemoryCache(store)
		if err != nil {
			logging.Error("Unable to initialise caching store: %v", err)
			return
		}
		store = cachedStore
	}
	storeCounter, err := sqlstore.NewCounterStore(config.DB)
	if err != nil {
		logging.Error("Unable to initialize performance counters: %v", err)
		return
	}
	metrics.DefaultCoreCounters.Update(storeCounter)
	store = counters.NewCounterWrapperStore(store)
	if config.EnableLocalOutputs {
		output.DisableLocalhostChecks()
	}

	if config.LaunchDataStorage {
		logging.Info("Launching embedded data store server with parameters %+v", config.DataStorage)
		srv, err := magpie.NewDataServer(config.DataStorage.SQL)
		if err != nil {
			logging.Error("Unable to create embedded data server: %v", err)
			return
		}
		endpoint, err := magpie.StartServer(srv, config.DataStorage.GRPC)
		if err != nil {
			logging.Error("Unable to start local data server: %v", err)
			return
		}
		config.GRPCDataStore.ServerEndpoint = endpoint
		config.GRPCDataStore.CAFile = config.DataStorage.GRPC.CertFile
		config.GRPCDataStore.TLS = config.DataStorage.GRPC.TLS
	}

	hordeserver := newServer(config.GRPCDataStore)
	mgr := output.NewLocalManager()

	config.Connect.SetSessionStoreConfig(config.DB.Type, config.DB.ConnectionString)

	downstreamStore, err := sqlstore.NewDownstreamStore(store, config.DB, uint8(config.DataCenterID), uint16(config.WorkerID))
	if err != nil {
		logging.Error("Error creating downstream message store: %v", err)
		return
	}

	fwStore, err := fwimage.NewSQLStore(config.DB)
	if err != nil {
		logging.Error("Error creating firmware image store: %v", err)
		return
	}

	var radiusServer radius.Server
	// Fire up RADIUS Server
	var apnStore storage.APNStore
	apnStore, err = sqlstore.NewSQLAPNStore(config.DB.Type, config.DB.ConnectionString, config.DB.CreateSchema)
	if err != nil {
		return
	}

	if config.Caching {
		cachedStore, err := memdb.PrimeAPNCache(apnStore)
		if err != nil {
			return
		}
		apnStore = cachedStore
	}

	apnConfig, err := storage.NewAPNCache(apnStore)
	if err != nil {
		logging.Error("Error creating APN cache: %v", err)
		return
	}
	if config.EmbeddedRADIUS {
		if err := apn.StartLocalRADIUS(config.RADIUS, store, apnStore, apnConfig); err != nil {
			logging.Error("Error starting RADIUS server: %v", err)
			return
		}
	} else {
		svr, err := apn.StartRADIUSgRPC(config.RADIUSGrpc, store, apnStore, apnConfig)
		if err != nil {
			logging.Error("Error launching RADIUS gRPC service: %v", err)
			return
		}
		logging.Info("RADIUS gRPC service runs at %s", svr.Endpoint())
	}

	publisher := make(chan model.DataMessage)
	rxtxReceiver := apn.NewRxTxReceiver(apnConfig, store, apnStore, downstreamStore, publisher)
	if err := fota.SetupFOTA(config.FOTA, rxtxReceiver, store, fwStore); err != nil {
		return
	}

	listenerGrpc, err := grpcutil.NewGRPCServer(config.RxTxGRPC)
	if err != nil {
		logging.Error("Error creating gRPC server for rxtx listener: %v", err)
		return
	}
	if err := listenerGrpc.Launch(func(s *grpc.Server) {
		rxtx.RegisterRxtxServer(s, rxtxReceiver)
	}, 250*time.Millisecond); err != nil {
		logging.Error("Error launching rxtx listener server: %v", err)
		return
	}

	logging.Info("RxTx service started on %s", listenerGrpc.Endpoint())
	if config.EmbeddedListener {
		rxtxServerParam := grpcutil.GRPCClientParam{ServerEndpoint: listenerGrpc.Endpoint()}
		conn, err := grpcutil.NewGRPCClientConnection(rxtxServerParam)
		if err != nil {
			logging.Error("Error connecting to rxtx server: %v", err)
			return
		}
		cl := deviceio.NewCoAPServer(rxtx.NewRxtxClient(conn), config.EmbeddedCOAP)
		if err := cl.Start(); err != nil {
			logging.Error("Error creating CoAP listener: %v", err)
			return
		}
		defer cl.Stop()

		conn, err = grpcutil.NewGRPCClientConnection(rxtxServerParam)
		if err != nil {
			logging.Error("Error connecting to rxtx server: %v", err)
			return
		}
		ul := deviceio.NewUDPListener(rxtx.NewRxtxClient(conn), config.EmbeddedUDP)
		if err := ul.Start(); err != nil {
			logging.Error("Error connecting to rxtx server: %v", err)
			return
		}
		defer ul.Stop()
		logging.Info("Started embedded UDP and CoAP listener")
	}
	api := restapi.NewServer(config.HTTP, config.GRPCDataStore, config.Connect,
		config.Github, store, fwStore, &messageSender{rxtxReceiver}, mgr, config.DeviceFieldMask)

	// Fire up Horde server
	if err := hordeserver.Start(store, api, publisher, mgr, config.DeviceFieldMask.ForcedFields()); err != nil {
		logging.Error("Unable to launch Horde main server: %v", err)
		return
	}

	monitoring, err := metrics.NewMonitoringServer(config.MonitoringEndpoint)
	if err != nil {
		logging.Error("Unable to create monitoring endpoint: %v", err)
		return
	}
	if err := monitoring.Start(); err != nil {
		logging.Error("Unable to start monitoring server: %v", err)
		return
	}

	if err := StartHordeManagementInterface(config.Management, apnStore, store, apnConfig); err != nil {
		logging.Error("Unable to start the management interface: %v", err)
		return
	}
	defer func() {
		if radiusServer != nil {
			logging.Info("Stopping RADIUS service")
			radiusServer.Stop()
		}

		hordeserver.Stop()
		logging.Info("Horde service is stopped")

		monitoring.Shutdown()
	}()
	logging.Debug("Ready. Waiting for server to terminate")
	utils.WaitForSignal()
}
