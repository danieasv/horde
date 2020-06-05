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

// TestInviteStore tests an invite store implementation
func testInviteStore(e TestEnvironment, s storage.DataStore, t *testing.T) {
	// Generate an invite
	i1, err := model.NewInvite(e.U1.ID, e.T12.ID)
	if err != nil {
		t.Fatalf("Cannot create (in memory) invite: %v", err)
	}
	if err := s.CreateInvite(i1); err != nil {
		t.Fatalf("Unable to create invite: %v", err)
	}
	i2, err := model.NewInvite(e.U2.ID, e.T21.ID)
	if err != nil {
		t.Fatalf("Cannot create (in memory) invite 2: %v", err)
	}
	if err := s.CreateInvite(i2); err != nil {
		t.Fatalf("Unable to create invite: %v", err)
	}

	// Private teams can't get invites
	ix1, _ := model.NewInvite(e.U1.ID, e.T1.ID)
	if err := s.CreateInvite(ix1); err == nil {
		t.Fatal("Expected error when creating invite for private team")
	}

	// Must be admin to create invite
	ix1, _ = model.NewInvite(e.U1.ID, e.T21.ID)
	if err := s.CreateInvite(ix1); err == nil {
		t.Fatal("Expected error when creating invite for non-admin team")
	}

	// Can't create the same invite twice
	if err := s.CreateInvite(i1); err != storage.ErrAlreadyExists {
		t.Fatal("Execpted ErrAlreadyExists but got ", err)
	}

	// Accept invite successfully

	// First -- create a new user that can accept invites
	tn := model.NewTeam()
	tn.ID = s.NewTeamID()
	tn.SetTag("name", "U1 team")

	un := model.NewUser(s.NewUserID(), "u3", model.AuthConnectID, tn.ID)
	tn.AddMember(model.NewMember(un, model.AdminRole))
	un.Name = "user # 3"
	if err := s.CreateUser(un, tn); err != nil {
		t.Fatal("Unable to create user #3 in test environment: ", err)
	}

	if err := s.AcceptInvite(i1, un.ID); err != nil {
		t.Fatal("Unable to accept invite #1: ", err)
	}

	// Can't accept invite for team you already are a member of
	if err := s.AcceptInvite(i2, e.U2.ID); err != storage.ErrAlreadyExists {
		t.Fatal("Expected ErrAlreadyExists but got ", err)
	}

	if err := s.AcceptInvite(i2, un.ID); err != nil {
		t.Fatal("Unable to accept invite #2: ", err)
	}

	i2.Code = i1.Code
	if err := s.AcceptInvite(i2, un.ID); err != storage.ErrNotFound {
		t.Fatal("Should not be able to accept twice: ", err)
	}

	// Delete existing invite
	ei, _ := model.NewInvite(e.U1.ID, e.T12.ID)
	s.CreateInvite(ei)
	if err := s.DeleteInvite(ei.Code, ei.TeamID, ei.UserID); err != nil {
		t.Fatalf("Did not expect error when deleting existing invite but got %v", err)
	}

	if err := s.DeleteInvite(i1.Code, i1.TeamID, i1.UserID); err != storage.ErrNotFound {
		t.Fatalf("Expected ErrNotFound but got %v", err)
	}
	ei, _ = model.NewInvite(e.U2.ID, e.T21.ID)
	s.CreateInvite(ei)
	if err := s.DeleteInvite(ei.Code, ei.TeamID, e.U1.ID); err != storage.ErrAccess {
		t.Fatalf("Expected ErrNoAccess when deleting invite without being admin but got %v", err)
	}
	s.DeleteInvite(ei.Code, ei.TeamID, ei.UserID)
	// Create several invites -- retrieve the list
	for i := 0; i < 5; i++ {
		tmp, err := model.NewInvite(e.U2.ID, e.T21.ID)
		if err != nil {
			t.Fatalf("Unable to create invite # %v: %v ", i, err)
		}
		s.CreateInvite(tmp)
	}

	if _, err := s.ListInvites(e.T21.ID, e.U1.ID); err == nil {
		t.Fatalf("Should not be able to retrieve invites when not admin but no error returned")
	}
	list, err := s.ListInvites(e.T21.ID, e.U2.ID)
	if err != nil {
		t.Fatal("Unable to retrieve list of invites: ", err)
	}
	for i, v := range list {
		invite, err := s.RetrieveInvite(v.Code)
		if err != nil {
			t.Fatalf("Unable to retrieve invite # %v: %v", i, err)
		}
		if !reflect.DeepEqual(invite, v) {
			t.Fatalf("Retrieved invite is different: %+v != %+v", invite, v)
		}
	}

	if _, err := s.RetrieveInvite("foo"); err != storage.ErrNotFound {
		t.Fatalf("Expected NotFound but got %v", err)
	}
	if _, err := s.RetrieveInvite(""); err != storage.ErrNotFound {
		t.Fatalf("Expected NotFound but got %v", err)
	}
}
