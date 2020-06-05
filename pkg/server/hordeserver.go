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
	"time"

	"github.com/eesrc/horde/pkg/api/apitoolbox"
	"github.com/eesrc/horde/pkg/metrics"
	"github.com/eesrc/horde/pkg/storage"
	"github.com/eesrc/horde/pkg/utils/grpcutil"

	"github.com/eesrc/horde/pkg/addons/magpie/datastore"

	"github.com/ExploratoryEngineering/logging"
	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/output"
	"github.com/eesrc/horde/pkg/restapi"
)

const storeChannelLength = 100

// NewServer creates a new Horde server
func newServer(cfg grpcutil.GRPCClientParam) *hordeServer {
	ret := &hordeServer{}
	if cfg.ServerEndpoint == "" {
		logging.Info("Device data client is disabled")
		return ret
	}
	conn, err := grpcutil.NewGRPCClientConnection(cfg)
	if err != nil {
		logging.Info("Unable to create device data client: %v", err)
		return ret
	}
	ret.dataStoreClient = datastore.NewDataStoreClient(conn)
	ret.storeChannel = make(chan model.DataMessage, storeChannelLength)
	go ret.storeData()
	logging.Info("Device data client connected to %v", cfg.ServerEndpoint)
	return ret
}

type hordeServer struct {
	restAPI          restapi.Server
	mgr              output.Manager
	upstreamMessages <-chan model.DataMessage
	store            storage.DataStore
	storeChannel     chan model.DataMessage
	dataStoreClient  datastore.DataStoreClient
}

func (h *hordeServer) Start(store storage.DataStore,
	restAPI restapi.Server,
	msg <-chan model.DataMessage,
	mgr output.Manager,
	systemFieldMask model.FieldMask) error {
	// Launch UDP listener, launch REST API, launch output manager
	h.restAPI = restAPI
	h.upstreamMessages = msg
	h.mgr = mgr
	h.store = store

	outputs, err := h.store.OutputListAll()
	if err != nil {
		return err
	}

	if err := h.restAPI.Start(); err != nil {
		return err
	}

	h.mgr.Refresh(outputs, systemFieldMask)
	go h.upstreamForwarder()
	return nil
}

// makeMetadata converts the message's device into binary metadata
// TODO(stalehd): Rewrite into sane format
func makeMetadata(msg model.DataMessage) []byte {
	ma := apitoolbox.JSONMarshaler()
	odm := apitoolbox.NewOutputDataMessageFromModel(msg, model.Collection{})
	buf, err := ma.MarshalToString(odm)
	if err != nil {
		return []byte{}
	}
	return []byte(buf)
}

func (h *hordeServer) storeData() {
	if h.dataStoreClient == nil {
		return
	}

	var sequence = int64(1)
	for msg := range h.storeChannel {
		dataMsg := &datastore.DataMessage{
			Sequence:     sequence,
			CollectionId: msg.Device.CollectionID.String(),
			DeviceId:     msg.Device.ID.String(),
			Payload:      msg.Payload,
			Created:      msg.Received.UnixNano(),
			Metadata:     makeMetadata(msg),
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*500)
		logging.Debug("Storing message (type:%s, payload=%d bytes)", msg.Transport.String(), len(msg.Payload))
		_, err := h.dataStoreClient.StoreData(ctx, dataMsg)
		if err != nil {
			logging.Warning("Got error calling PutData: %v", err)
		}
		cancel()
		sequence++
	}
}

func (h *hordeServer) upstreamForwarder() {
	for msg := range h.upstreamMessages {
		h.mgr.Publish(msg)
		select {
		case h.storeChannel <- msg:
			// ok
		default:
			logging.Warning("Store channel dropped message.")
		}
		metrics.DefaultCoreCounters.MessagesInCount.Add(1)
	}
}

func (h *hordeServer) Stop() error {
	if err := h.restAPI.Stop(); err != nil {
		return err
	}
	h.mgr.Shutdown()
	return nil
}
