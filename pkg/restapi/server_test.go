package restapi

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
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/eesrc/horde/pkg/api"
	"github.com/eesrc/horde/pkg/ghlogin"
	"github.com/eesrc/horde/pkg/utils/grpcutil"

	"github.com/TelenorDigital/goconnect"
	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/output"
	"github.com/eesrc/horde/pkg/storage"
	"github.com/eesrc/horde/pkg/storage/fwimage"
	"github.com/eesrc/horde/pkg/storage/sqlstore"
)

var testParams = ServerParameters{
	Endpoint:    "localhost:0",
	ACME:        ACMEParameters{Enabled: false},
	TLSCertFile: "",
	TLSKeyFile:  "",
}

var connectParams = ConnectIDParameters{
	Enabled: true,
}
var ghParams = ghlogin.Config{
	DBDriver:           "sqlite3",
	DBConnectionString: ":memory:",
}
var dataClientParams grpcutil.GRPCClientParam

var mask = model.FieldMaskParameters{Forced: "", Default: ""}

var imageStore = fwimage.NewFileSystemStore("./images")

func newDummyMessageSender() *dummySender {
	return &dummySender{}
}

type dummySender struct {
	FailOnIMSI int64
}

func (d *dummySender) Send(dev model.Device, m model.DownstreamMessage) error {
	if d.FailOnIMSI == dev.IMSI {
		return errors.New("Send failed")
	}
	return nil
}

func TestServer(t *testing.T) {

	server := NewServer(testParams, dataClientParams, connectParams,
		ghParams, sqlstore.NewMemoryStore(), imageStore, newDummyMessageSender(),
		output.NewDummyManager(), mask)

	go server.Start()

	time.Sleep(100 * time.Millisecond)
	defer server.Stop()
}

func TestRootHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "http://example.com/", nil)
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()

	s := NewServer(testParams, dataClientParams, connectParams,
		ghParams, sqlstore.NewMemoryStore(), imageStore, &dummySender{},
		output.NewDummyManager(), mask)
	server := s.(*restServer)
	server.rootHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Exptected OK but got %d from root handler", rec.Code)
	}

}

func TestConnectIntegration(t *testing.T) {
	store := sqlstore.NewMemoryStore()
	s := NewServer(testParams, dataClientParams, connectParams,
		ghParams, store, imageStore, &dummySender{}, output.NewDummyManager(), mask)
	if s == nil {
		t.Fatal("Couldn't create server")
	}
	server := s.(*restServer)
	sess1 := goconnect.Session{
		UserID:        "connect-1",
		Email:         "test@example.com",
		Phone:         "12345678",
		VerifiedEmail: false,
		VerifiedPhone: false,
		Name:          "Test User 1",
	}
	user1 := server.addOrUpdateConnectUser(sess1)
	if user1 == nil {
		t.Fatal("Did not get a proper user in return")
	}
	if user1.ExternalID != sess1.UserID {
		t.Fatal("Did not get the right user")
	}

	sess2 := sess1
	sess2.VerifiedEmail = true
	sess2.VerifiedPhone = true
	sess2.Phone = "99887766"
	sess2.Name = "Other name"
	user2 := server.addOrUpdateConnectUser(sess2)
	if user1.ID != user2.ID {
		t.Fatal("User has changed ID")
	}
	if user2.Name != sess2.Name {
		t.Fatal("Name isn't updated")
	}

	storedUser, err := store.RetrieveUserByExternalID(sess1.UserID, model.AuthConnectID)
	if err != nil {
		t.Fatal("Could not retrieve connect ID user: ", err)
	}
	if *storedUser != *user2 {
		t.Fatalf("User isn't stored correctly %+v != %+v", user2, storedUser)
	}
}

// Test the connect ID emulator code (just because OCD)
func TestConnectEmulator(t *testing.T) {
	store := sqlstore.NewMemoryStore()
	s := NewServer(testParams, dataClientParams, connectParams, ghParams,
		store, imageStore, &dummySender{}, output.NewDummyManager(), mask)
	server := s.(*restServer)
	f := server.emulateConnect(func(w http.ResponseWriter, r *http.Request) {
		// empty
	})

	r := httptest.NewRequest("GET", "/foO", nil)
	w := httptest.NewRecorder()
	f.ServeHTTP(w, r)
}

func makeTestUser(ms storage.DataStore, t *testing.T) model.User {
	// Create connect ID user, put in store, set session accordingly
	team := model.NewTeam()
	team.ID = model.TeamKey(1)
	connectUser := model.NewUser(model.UserKey(1), "1", model.AuthConnectID, team.ID)
	connectUser.Email = "doe@example.com"
	connectUser.Name = "John Doe"
	connectUser.VerifiedEmail = true
	if err := ms.CreateUser(connectUser, team); err != nil && err != storage.ErrAlreadyExists {
		t.Fatal("Unable to store user: ", err)
	}
	return connectUser
}

func makeSessionContext(ctx context.Context, connectUser model.User) context.Context {
	session := goconnect.Session{
		UserID:        connectUser.ExternalID,
		Name:          connectUser.Name,
		Email:         connectUser.Email,
		VerifiedEmail: connectUser.VerifiedEmail}
	return context.WithValue(ctx, goconnect.SessionContext, session)
}

// Ensure Connect ID sessions are properly converted to users
func TestConnectUserConversion(t *testing.T) {
	store := sqlstore.NewMemoryStore()
	s := NewServer(testParams, dataClientParams, connectParams, ghParams,
		store, imageStore, &dummySender{}, output.NewDummyManager(), mask)
	server := s.(*restServer)
	var req *http.Request
	f := server.authSessionToUserHandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Must receive user
		req = r
		t.Logf("Context: %+v", r.Context())
	})

	r := httptest.NewRequest("GET", "/foO", nil)
	if user := server.UserFromRequest(nil); user != nil {
		t.Fatal("Should not get a connect user for a nil request")
	}
	if user := server.UserFromRequest(r); user != nil {
		t.Fatal("Should not get a connect user for a blank session")
	}
	if user := server.UserFromRequest(r.WithContext(context.Background())); user != nil {
		t.Fatal("Should not get a connect user for a blank session")
	}
	if user := server.UserFromRequest(r.WithContext(context.WithValue(r.Context(), api.UserKey, 1))); user != nil {
		t.Fatal("Should not get a connect user for a blank session")
	}
	connectUser := makeTestUser(store, t)
	w := httptest.NewRecorder()
	sessionContext := makeSessionContext(r.Context(), connectUser)
	f.ServeHTTP(w, r.WithContext(sessionContext))

	user := server.UserFromRequest(req)
	if user == nil {
		t.Fatal("Expected user to resolve")
	}
	if user.ID != connectUser.ID {
		t.Fatal("Did not resolve to the correct user")
	}
}
