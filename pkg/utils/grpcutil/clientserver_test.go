package grpcutil

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
	"testing"
	"time"

	magpie "github.com/eesrc/horde/pkg/addons/magpie/datastore"
	"google.golang.org/grpc"
)

type dummyServer struct {
}

func (d *dummyServer) PutData(magpie.DataStore_PutDataServer) error {
	return nil
}

func (d *dummyServer) GetData(*magpie.DataFilter, magpie.DataStore_GetDataServer) error {
	return nil
}

func (d *dummyServer) GetDataMetrics(context.Context, *magpie.DataFilter) (*magpie.DataMetrics, error) {
	return nil, nil
}
func (d *dummyServer) StoreData(ctx context.Context, m *magpie.DataMessage) (*magpie.Receipt, error) {
	return &magpie.Receipt{Sequence: m.Sequence}, nil
}
func TestClientAndServer(t *testing.T) {
	serverConfig := GRPCServerParam{"127.0.0.1:0", false, "", ""}

	server, err := NewGRPCServer(serverConfig)
	if err != nil {
		t.Fatal(err)
	}

	if server.Launch(func(s *grpc.Server) {
		magpie.RegisterDataStoreServer(s, &dummyServer{})
	}, 200*time.Millisecond); err != nil {
		t.Fatal(err)
	}

	defer server.Stop()

	clientConfig := GRPCClientParam{server.Endpoint(), false, "", ""}

	conn, err := NewGRPCClientConnection(clientConfig)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

}

func TestInvalidServerTLSConfig(t *testing.T) {
	serverConfig := GRPCServerParam{"localhost:0", true, "", ""}
	server, err := NewGRPCServer(serverConfig)
	if err != nil {
		t.Fatal(err)
	}
	if err := server.Start(nil); err == nil {
		t.Fatal("Expected error with TLS flag but no parameters")
	}
}

func TestInvalidClientTLSConfig(t *testing.T) {
	clientConfig := GRPCClientParam{"127.0.0.1:0", true, "", ""}
	if _, err := NewGRPCClientConnection(clientConfig); err == nil {
		t.Fatal("Expected error with incorrect TLS flag but no error returned")
	}
}

func TestInvalidClientTLSFiles(t *testing.T) {
	clientConfig := GRPCClientParam{"127.0.0.1:0", true, "foo.ca", ""}
	if _, err := NewGRPCClientConnection(clientConfig); err == nil {
		t.Fatal("Expected error with missing CA file")
	}
}

func TestInvalidServerTLSFiles(t *testing.T) {
	serverConfig := GRPCServerParam{"localhost:0", true, "invalid.ca", "invalid.key"}
	srv, err := NewGRPCServer(serverConfig)
	if err != nil {
		t.Fatal(err)
	}
	if err := srv.Start(nil); err == nil {
		t.Fatal("Expected error with missing cert and key but no error returned")
	}
}
