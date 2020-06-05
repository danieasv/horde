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
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/TelenorDigital/goconnect"
	"github.com/eesrc/horde/pkg/addons/magpie"
	"github.com/eesrc/horde/pkg/addons/magpie/datastore"
	"github.com/eesrc/horde/pkg/api/apipb"
	"github.com/eesrc/horde/pkg/ghlogin"
	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/output/outputconfig"
	"github.com/eesrc/horde/pkg/storage/sqlstore"
	"github.com/eesrc/horde/pkg/utils/grpcutil"
	"github.com/eesrc/horde/pkg/version"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var dataStoreSvr datastore.DataStoreServer
var dataStoreClient datastore.DataStoreClient

func init() {
	var err error
	dataStoreSvr, err = magpie.NewDataServer(sqlstore.Parameters{
		ConnectionString: "file::memory:?cache=shared",
		Type:             "sqlite3",
		CreateSchema:     true,
	})
	if err != nil {
		panic(err.Error())
	}
	ep, err := magpie.StartServer(dataStoreSvr, grpcutil.GRPCServerParam{
		Endpoint: "127.0.0.1:0",
	})
	if err != nil {
		panic(err.Error())
	}
	conn, err := grpcutil.NewGRPCClientConnection(grpcutil.GRPCClientParam{
		ServerEndpoint: ep,
	})
	if err != nil {
		panic(err.Error())
	}
	dataStoreClient = datastore.NewDataStoreClient(conn)
}

func TestSystemServiceInfo(t *testing.T) {
	assert := require.New(t)

	version.BuildDate = "2020-01-01T12:00"
	version.Name = "the-name"
	version.Number = "1.2.3"

	store := sqlstore.NewMemoryStore()

	svc := newSystemService(model.FieldMaskParameters{Forced: "location", Default: ""}, store, dataStoreClient)
	assert.NotNil(svc)

	resp, err := svc.GetSystemInfo(context.Background(), &apipb.SystemInfoRequest{})
	assert.NoError(err)
	assert.NotNil(resp)

	assert.Equal(version.BuildDate, resp.BuildDate.Value)
	assert.Equal(version.Name, resp.ReleaseName.Value)
	assert.Equal(version.Number, resp.Version.Value)
}

func TestSystemServiceDump(t *testing.T) {
	assert := require.New(t)

	store := sqlstore.NewMemoryStore()

	svc := newSystemService(model.FieldMaskParameters{Forced: "", Default: ""}, store, dataStoreClient)
	assert.NotNil(svc)

	// Attempt unauthenticated request - should return error
	_, err := svc.DataDump(context.Background(), &apipb.DataDumpRequest{})
	assert.Error(err)
	assert.Equal(codes.Unauthenticated.String(), status.Code(err).String())

	user, _, ctx := createAuthenticatedContext(assert, store)

	// Initial data dump without any data will return just the profile
	resp, err := svc.DataDump(ctx, &apipb.DataDumpRequest{})
	assert.NoError(err)
	assert.NotNil(resp)
	assert.Len(resp.Collections, 0)

	wg := &sync.WaitGroup{}
	wg.Add(50)

	// Populate store with collections, devices, outputs, data for devices
	for i := 0; i < 10; i++ {
		c := model.NewCollection()
		c.ID = store.NewCollectionID()
		c.SetTag("name", fmt.Sprintf("Collection %d", i))
		c.TeamID = user.PrivateTeamID
		c.Firmware.Management = model.CollectionManagement
		assert.NoError(store.CreateCollection(user.ID, c))

		f := model.NewFirmware()
		f.ID = store.NewFirmwareID()
		f.CollectionID = c.ID
		f.SHA256 = fmt.Sprintf("%d", i)
		f.Version = fmt.Sprintf("%d", i)
		assert.NoError(store.CreateFirmware(user.ID, f))

		o := model.NewOutput()
		o.ID = store.NewOutputID()
		o.CollectionID = c.ID
		o.Type = "udp"
		o.Config[outputconfig.UDPHost] = "example.com"
		o.Config[outputconfig.UDPPort] = 4711
		o.Enabled = false
		assert.NoError(store.CreateOutput(user.ID, o))

		for j := 0; j < 5; j++ {
			d := model.NewDevice()
			d.ID = store.NewDeviceID()
			d.CollectionID = c.ID
			d.SetTag("name", fmt.Sprintf("Device %d/%d", i, j))
			d.IMEI = int64(100000*i + j)
			d.IMSI = d.IMEI
			assert.NoError(store.CreateDevice(user.ID, d))

			pc, err := dataStoreClient.PutData(context.Background())
			assert.NoError(err)
			go func() {
				count := 0
				for {
					m, err := pc.Recv()
					if err != nil {
						t.Logf("Ack has completed err = %v", err)
						wg.Done()
						return
					}
					t.Logf("Gots ack %+v", m)
					count++
				}
			}()
			for k := 0; k < 5; k++ {
				assert.NoError(pc.Send(&datastore.DataMessage{
					Sequence:     int64(k),
					CollectionId: c.ID.String(),
					DeviceId:     d.ID.String(),
					Created:      time.Now().UnixNano(),
					Metadata:     []byte(`{"transport":"udp"}`),
					Payload:      []byte(fmt.Sprintf("hello there # %d", k)),
				}))
			}
			pc.CloseSend()
		}
	}
	wg.Wait()
	// Repeat request - this should dump data
	resp, err = svc.DataDump(ctx, &apipb.DataDumpRequest{})
	assert.NoError(err)
	assert.NotNil(resp)
	assert.Len(resp.Collections, 10)
	for _, v := range resp.Collections {
		assert.Len(v.Outputs, 1)
		//assert.Len(v.Firmwares, 10)
		assert.Len(v.Devices, 5)
		for _, d := range v.Devices {
			assert.Len(d.Data, 5)
		}
	}
	assert.Len(resp.Teams, 1)
}

func TestSystemServiceProfile(t *testing.T) {
	assert := require.New(t)
	store := sqlstore.NewMemoryStore()

	svc := newSystemService(model.FieldMaskParameters{Forced: "", Default: ""}, store, dataStoreClient)
	assert.NotNil(svc)

	// Retrieve unauthenticated
	_, err := svc.GetUserProfile(context.Background(), &apipb.UserProfileRequest{})
	assert.Error(err)
	assert.Equal(codes.Unauthenticated.String(), status.Code(err).String())

	// Get profile for internal user
	user, token := createUserAndToken(assert, model.AuthInternal, store)
	md := metadata.New(map[string]string{tokenHeaderName: token})
	ctxInternal := metadata.NewIncomingContext(context.Background(), md)

	res, err := svc.GetUserProfile(ctxInternal, &apipb.UserProfileRequest{})
	assert.NoError(err)
	assert.NotNil(res)
	assert.Equal("internal", res.Provider.Value)

	// Get profile for connect id user
	user, _ = createUserAndToken(assert, model.AuthConnectID, store)
	session := goconnect.Session{
		UserID: user.ExternalID,
		Email:  "johndoe@example.com",
	}
	ctxConnect := context.WithValue(context.Background(), goconnect.SessionContext, session)
	res, err = svc.GetUserProfile(ctxConnect, &apipb.UserProfileRequest{})
	assert.NoError(err)
	assert.NotNil(res)
	assert.Equal("connect", res.Provider.Value)

	// Get profile for github user
	user, _ = createUserAndToken(assert, model.AuthGitHub, store)
	profile := ghlogin.Profile{
		Login: user.ExternalID,
	}
	ctxGithub := context.WithValue(context.Background(), ghlogin.GitHubSessionProfile, profile)
	res, err = svc.GetUserProfile(ctxGithub, &apipb.UserProfileRequest{})
	assert.NoError(err)
	assert.NotNil(res)
	assert.Equal("github", res.Provider.Value)

}
