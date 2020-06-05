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
	"testing"

	"github.com/eesrc/horde/pkg/storage"
)

// StorageTest ensures the implementation of the DataStore interface
// is done properly
func StorageTest(t *testing.T, s storage.DataStore) {
	e := NewTestEnvironment(t, s)
	testUserStore(s, t)
	testTeamStore(e, s, t)
	testTokenStore(e, s, t)
	testCollectionStore(e, s, t)
	testDeviceStore(e, s, t)
	testOutputStore(e, s, t)
	testInviteStore(e, s, t)

	testUserUpdates(e, s, t)

	testFirmwareStore(e, s, t)
}

func testUserUpdates(e TestEnvironment, s storage.DataStore, t *testing.T) {

	e.U1.Email = "test@example.com"
	e.U1.Name = "Name"
	e.U1.Phone = "555 12345"
	if err := s.UpdateUser(&e.U1); err != nil {
		t.Fatal(err)
	}

	teamListA, err := s.ListTeams(e.U1.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range teamListA {
		for _, m := range v.Members {
			if m.User.ID == e.U1.ID {
				if m.User.Email != e.U1.Email || m.User.Name != e.U1.Name || m.User.Phone != e.U1.Phone {
					t.Fatalf("User 1 is incorrectly listed in team list#1: User: %+v Member: %+v", e.U1, m.User)
				}
			}
		}
	}

	e.U1.Name = "Updated name"
	if err := s.UpdateUser(&e.U1); err != nil {
		t.Fatal(err)
	}

	teamListB, err := s.ListTeams(e.U1.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range teamListB {
		for _, m := range v.Members {
			if m.User.ID == e.U1.ID {
				if m.User.Email != e.U1.Email || m.User.Name != e.U1.Name || m.User.Phone != e.U1.Phone {
					t.Fatalf("User 1 is incorrectly listed in team list#2: User: %+v Member: %+v", e.U1, m.User)
				}
			}
		}
	}
}
