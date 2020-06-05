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
	"encoding/json"
	"testing"
	"time"

	"github.com/eesrc/horde/pkg/addons/magpie/datastore"
	"github.com/eesrc/horde/pkg/storage/sqlstore"
	"github.com/eesrc/horde/pkg/utils/grpcutil"
)

type metadataStruct struct {
	Something string                 `json:"someThing"`
	Other     string                 `json:"other"`
	Value     int                    `json:"value"`
	Tags      map[string]interface{} `json:"tags"`
}

func TestServer(t *testing.T) {
	rpcCfg := grpcutil.GRPCServerParam{TLS: false,
		CertFile: "",
		KeyFile:  "",
		Endpoint: "localhost:0"}
	dbCfg := sqlstore.Parameters{
		Type:             "sqlite3",
		ConnectionString: ":memory:",
		CreateSchema:     true,
	}

	server, err := NewDataServer(dbCfg)
	if err != nil {
		t.Fatal(err)
	}

	ep, err := StartServer(server, rpcCfg)
	if err != nil {
		t.Fatal(err)
	}

	// Ugh. This is ugly and I *know* it will fail on a build server but
	// give the server some time to get up and running.
	time.Sleep(100 * time.Millisecond)

	// Generate some data
	conn, err := grpcutil.NewGRPCClientConnection(grpcutil.GRPCClientParam{
		TLS:                false,
		CAFile:             "",
		ServerHostOverride: "",
		ServerEndpoint:     ep,
	})
	if err != nil {
		t.Fatal(err)
	}
	client := datastore.NewDataStoreClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pdc, err := client.PutData(ctx)
	if err != nil {
		t.Fatal(err)
	}
	completed := make(chan int64)
	go func() {
		for {
			msg, err := pdc.Recv()
			if err != nil {
				return
			}
			completed <- msg.Sequence
		}
	}()
	ts := time.Now().UnixNano()
	first := ts

	const numPackets = 10
	for i := 0; i < numPackets; i++ {
		metadata, err := json.Marshal(&metadataStruct{Value: i})
		if err != nil {
			t.Fatal(err)
		}
		n := byte(i)
		if err := pdc.Send(&datastore.DataMessage{
			Sequence:     int64(i),
			CollectionId: "100",
			DeviceId:     "100",
			Created:      ts,
			Metadata:     metadata,
			Payload:      []byte{n, n, n, n},
		}); err != nil {
			t.Fatal(err)
		}
		ts += int64(time.Second)
	}
	last := ts - int64(time.Second)

	for i := 0; i < numPackets; i++ {
		select {
		case <-completed:
		case <-time.After(50 * time.Millisecond):
			t.Fatalf("No response from server after %d packets", i)
		}
	}
	if err := pdc.CloseSend(); err != nil {
		t.Fatal(err)
	}

	// Metrics should match min/max

	m, err := client.GetDataMetrics(ctx, &datastore.DataFilter{
		CollectionId: "100",
		DeviceId:     "100",
	})
	if err != nil {
		t.Fatal(err)
	}
	if m.FirstDataPoint != first {
		t.Fatalf("Expected first to be %d but it was %d", first, m.FirstDataPoint)
	}
	if m.LastDataPoint != last {
		t.Fatalf("Expected last to be %d but it was %d", last, m.LastDataPoint)
	}
	if m.MessageCount != numPackets {
		t.Fatalf("Expected message count to be %d but it was %d", numPackets, m.MessageCount)
	}

	getdata, err := client.GetData(ctx, &datastore.DataFilter{
		CollectionId: "100",
		DeviceId:     "100",
		From:         first,
		To:           last,
		Limit:        numPackets * 2,
	})

	if err != nil {
		t.Fatal(err)
	}

	received := 0
	for {
		var m datastore.DataMessage
		if err := getdata.RecvMsg(&m); err != nil {
			break
		}
		received++
		t.Logf("Got %v ", err)
	}

	if received != numPackets {
		t.Fatalf("Expected %d messages returned but got %d", numPackets, received)
	}
	getdata.CloseSend()
}

func BenchmarkServer(b *testing.B) {
	rpcCfg := grpcutil.GRPCServerParam{TLS: false,
		CertFile: "",
		KeyFile:  "",
		Endpoint: "localhost:0"}
	dbCfg := sqlstore.Parameters{
		Type:             "postgres",
		ConnectionString: "postgres://localhost/horde?sslmode=disable",
		CreateSchema:     true,
	}
	server, err := NewDataServer(dbCfg)
	if err != nil {
		b.Fatal(err)
	}

	ep, err := StartServer(server, rpcCfg)
	if err != nil {
		b.Fatal(err)
	}
	b.Logf("Endpoint for server: %s", ep)

	// Generate some data

	conn, err := grpcutil.NewGRPCClientConnection(grpcutil.GRPCClientParam{
		TLS:                false,
		CAFile:             "",
		ServerHostOverride: "",
		ServerEndpoint:     ep,
	})
	if err != nil {
		b.Fatal(err)
	}
	client := datastore.NewDataStoreClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pdc, err := client.PutData(ctx)
	if err != nil {
		b.Fatal(err)
	}
	completed := make(chan int64)
	go func() {
		for {
			msg, err := pdc.Recv()
			if err != nil {
				return
			}
			completed <- msg.Sequence
		}
	}()
	ts := time.Now().UnixNano()

	for i := 0; i < b.N; i++ {
		n := byte(i)
		if err := pdc.Send(&datastore.DataMessage{
			Sequence:     int64(i),
			CollectionId: "100",
			DeviceId:     "100",
			Created:      ts,
			Payload:      []byte{n, n, n, n},
		}); err != nil {
			b.Fatal(err)
		}
		ts += int64(time.Second)
	}

	for i := 0; i < b.N; i++ {
		select {
		case <-completed:
		case <-time.After(50 * time.Millisecond):
			b.Fatalf("No response from server after %d packets", i)
		}
	}
	if err := pdc.CloseSend(); err != nil {
		b.Fatal(err)
	}

}
