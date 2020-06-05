package magpie

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
	"io"
	"time"

	"github.com/eesrc/horde/pkg/storage/sqlstore"

	"github.com/ExploratoryEngineering/logging"
	"github.com/eesrc/horde/pkg/addons/magpie/datastore"
	"github.com/eesrc/horde/pkg/metrics"
	"github.com/eesrc/horde/pkg/utils/grpcutil"
	"google.golang.org/grpc"
)

type backendStore interface {
	Store(data *datastore.DataMessage) error
	Query(filter *datastore.DataFilter) (chan datastore.DataMessage, error)
	Metrics(filter *datastore.DataFilter) (datastore.DataMetrics, error)
}

type dataServer struct {
	nest backendStore
}

// StartServer launches the gRPC service
func StartServer(srv datastore.DataStoreServer, cfg grpcutil.GRPCServerParam) (string, error) {
	server, err := grpcutil.NewGRPCServer(cfg)
	if err != nil {
		return "", err
	}

	if err := server.Launch(func(s *grpc.Server) {
		datastore.RegisterDataStoreServer(s, srv)
	}, 200*time.Millisecond); err != nil {
		return "", err
	}
	return server.Endpoint(), nil
}

// NewDataServer creates a new data server. It will return the server, the
// server's endpoint. If there's an error creating the server an error will be
// returned
func NewDataServer(cfg sqlstore.Parameters) (datastore.DataStoreServer, error) {
	sqlNest, err := newSQLStore(cfg)
	if err != nil {
		return nil, err
	}
	metrics.DefaultStoreCounters.Start()
	return &dataServer{nest: sqlNest}, nil
}

func (s *dataServer) PutData(stream datastore.DataStore_PutDataServer) error {
	for {
		data, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			logging.Debug("Got error receiving: %v", err)
			metrics.DefaultStoreCounters.Errors.Inc()
			return err
		}
		start := time.Now()
		if err := s.nest.Store(data); err != nil {
			logging.Error("Unable to store data: %v", err)
			metrics.DefaultStoreCounters.Errors.Inc()
			continue
		}
		metrics.DefaultStoreCounters.Stored.Inc()
		durationMs := float64(time.Since(start)) / float64(time.Millisecond)
		logging.Info("Took %6.3f ms to store %d bytes of data", durationMs, len(data.Payload))
		go func() {
			if err := stream.Send(&datastore.Receipt{Sequence: data.Sequence}); err != nil {
				logging.Error("error sending: %v", err)
			}
		}()
	}
}

func (s *dataServer) GetData(filter *datastore.DataFilter, stream datastore.DataStore_GetDataServer) error {

	start := time.Now()
	ch, err := s.nest.Query(filter)
	durationMs := float64(time.Since(start)) / float64(time.Millisecond)
	logging.Info("Took %6.3f ms to read data", durationMs)
	if err != nil {
		return err
	}
	for m := range ch {
		if err := stream.Send(&m); err != nil {
			if err == io.EOF {
				return nil
			}
		}
	}
	return nil
}

func (s *dataServer) GetDataMetrics(ctx context.Context, filter *datastore.DataFilter) (*datastore.DataMetrics, error) {
	m, err := s.nest.Metrics(filter)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (s *dataServer) StoreData(ctx context.Context, msg *datastore.DataMessage) (*datastore.Receipt, error) {
	if err := s.nest.Store(msg); err != nil {
		logging.Error("Unable to store data: %v", err)
		metrics.DefaultStoreCounters.Errors.Inc()
		// This is technically not a success but..,
		return nil, err
	}
	metrics.DefaultStoreCounters.Stored.Inc()
	return &datastore.Receipt{Sequence: msg.Sequence}, nil
}
