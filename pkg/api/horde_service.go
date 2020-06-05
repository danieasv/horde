package api

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
	"github.com/eesrc/horde/pkg/addons/magpie/datastore"
	"github.com/eesrc/horde/pkg/api/apipb"
	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/output"
	"github.com/eesrc/horde/pkg/storage"
)

type apiServer struct {
	collectionService
	deviceService
	firmwareService
	tokenService
	teamService
	outputService
	systemService
}

// NewHordeAPIService creates a new HordeAPIServer
func NewHordeAPIService(store storage.DataStore,
	fieldMask model.FieldMaskParameters,
	outputManager output.Manager,
	dataStoreClient datastore.DataStoreClient,
	messageSender DownstreamMessageSender, firmwareImageStore storage.FirmwareImageStore) apipb.HordeServer {
	return &apiServer{
		collectionService: newCollectionService(store, fieldMask, outputManager, dataStoreClient, messageSender),
		deviceService:     newDeviceService(store, dataStoreClient, messageSender),
		firmwareService:   newFirmwareService(store, firmwareImageStore),
		tokenService:      newTokenService(store),
		teamService:       newTeamService(store),
		outputService:     newOutputService(store, outputManager, fieldMask),
		systemService:     newSystemService(fieldMask, store, dataStoreClient),
	}
}
