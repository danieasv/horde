package storetest

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
	"reflect"
	"testing"

	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
)

// TestUserStore tests an implementation of the user store
func testUserStore(store storage.DataStore, t *testing.T) {
	team := model.NewTeam()
	team.ID = model.TeamKey(48993)
	user := model.NewUser(store.NewUserID(), "connectid-1", model.AuthConnectID, team.ID)
	user.Name = "John Doe"
	user.Email = "johndoe@example.com"
	user.Phone = "5544662211"
	user.VerifiedEmail = true
	user.VerifiedPhone = true

	if err := store.CreateUser(user, team); err != nil {
		t.Fatal("Unable to store new user: ", err)
	}

	// Create a duplicate. It shouldn't work
	u2 := model.NewUser(user.ID, "connectid-2", model.AuthConnectID, model.TeamKey(2))
	if err := store.CreateUser(u2, team); err == nil {
		t.Fatal("Shouldn't be able to store new user with existing user id but did")
	}

	// Create another duplicate for connect id
	u3 := model.NewUser(store.NewUserID(), user.ExternalID, model.AuthConnectID, model.TeamKey(2))
	if err := store.CreateUser(u3, team); err == nil {
		t.Fatalf("Shouldn't be able to store new user with existing connect id but did (user: %+v)", u3)
	}

	// Retrieve the first user based on the connect id
	u4, err := store.RetrieveUserByExternalID(user.ExternalID, model.AuthConnectID)
	if err != nil {
		t.Fatal("Unable to retrieve user: ", err)
	}
	if !reflect.DeepEqual(*u4, user) {
		t.Fatalf("Retrieved user isn't the same: %+v != %+v", user, u4)
	}

	if _, err := store.RetrieveUserByExternalID(u2.ExternalID, model.AuthConnectID); err != storage.ErrNotFound {
		t.Fatalf("Expected not found error but got %v", err)
	}

	if _, err := store.RetrieveUserByExternalID("", model.AuthConnectID); err != storage.ErrNotFound {
		t.Fatalf("Expected not found error but got %v", err)
	}

	u5 := *u4
	u5.Name = "Updated"
	if err := store.UpdateUser(&u5); err != nil {
		t.Fatal("Unable to update user: ", err)
	}

	unknown := model.NewUser(store.NewUserID(), "", model.AuthConnectID, model.TeamKey(1))
	if err := store.UpdateUser(&unknown); err != storage.ErrNotFound {
		t.Fatalf("Should not be able to update user (%+v) but did (err = %v)", unknown, err)
	}

	// TODO: Test user update if connect id already exists (should return ErrAlreadyExists)
}
