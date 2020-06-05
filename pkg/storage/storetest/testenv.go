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

	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
)

// TestEnvironment holds basics for the storage tests. They operate a lot on
// users and teams.
type TestEnvironment struct {
	U1  model.User
	U2  model.User
	U3  model.User
	T1  model.Team       // U1's private team
	T2  model.Team       // U2's private team
	T12 model.Team       // Team containing both U1 (admin) and U2 (member)
	T21 model.Team       // Team containing both U2 (admin) and U1 (member)
	T3  model.Team       // A team that doesn't contain U1 or U2 at all
	C1  model.Collection // U1's private collection
	C2  model.Collection // U2's private collection
	C12 model.Collection // Collection owned by team T12
	C21 model.Collection // Collection owned by team T21
	C3  model.Collection // Collection owned by team T3s
}

// NewTestEnvironment creates a new testing environment
func NewTestEnvironment(t *testing.T, s storage.DataStore) TestEnvironment {
	t1 := model.NewTeam()
	t1.ID = s.NewTeamID()
	t1.SetTag("name", "U1 team")

	t2 := model.NewTeam()
	t2.ID = s.NewTeamID()
	t2.SetTag("name", "U2 team")

	u1 := model.NewUser(s.NewUserID(), "c1", model.AuthConnectID, t1.ID)
	t1.AddMember(model.NewMember(u1, model.AdminRole))
	u1.Name = "user 1"
	u1.ExternalID = s.NewUserID().String()
	if err := s.CreateUser(u1, t1); err != nil {
		t.Fatal("Unable to create user #1 in test environment: ", err)
	}

	u2 := model.NewUser(s.NewUserID(), "u2", model.AuthConnectID, t2.ID)
	t2.AddMember(model.NewMember(u2, model.AdminRole))
	u2.Name = "user 2"
	u2.ExternalID = s.NewUserID().String()
	if err := s.CreateUser(u2, t2); err != nil {
		t.Fatal("Unable to create user #2 in test environment: ", err)
	}

	t12 := model.NewTeam()
	t12.ID = s.NewTeamID()
	t12.SetTag("name", "U1+U2 team")
	t12.AddMember(model.NewMember(u1, model.AdminRole))
	t12.AddMember(model.NewMember(u2, model.MemberRole))
	if err := s.CreateTeam(t12); err != nil {
		t.Fatal("Unable to create team #1 in test environment: ", err)
	}

	t21 := model.NewTeam()
	t21.ID = s.NewTeamID()
	t21.SetTag("name", "U2+U1 team")
	t21.AddMember(model.NewMember(u2, model.AdminRole))
	t21.AddMember(model.NewMember(u1, model.MemberRole))
	if err := s.CreateTeam(t21); err != nil {
		t.Fatal("Unable to create team #2 in test environment: ", err)
	}

	t3 := model.NewTeam()
	t3.ID = s.NewTeamID()
	t3.SetTag("name", "U3 team")

	u3 := model.NewUser(s.NewUserID(), "u3", model.AuthConnectID, t3.ID)
	t3.AddMember(model.NewMember(u3, model.AdminRole))
	u3.Name = "user 3"
	u3.ExternalID = s.NewUserID().String()
	if err := s.CreateUser(u3, t3); err != nil {
		t.Fatal("Unable to create user #3 in test environment: ", err)
	}

	c1 := model.NewCollection()
	c1.ID = s.NewCollectionID()
	c1.TeamID = t1.ID
	c1.SetTag("name", "U1 private collection")
	if err := s.CreateCollection(u1.ID, c1); err != nil {
		t.Fatal("Unable to create collection #1 in test environment: ", err)
	}

	c2 := model.NewCollection()
	c2.ID = s.NewCollectionID()
	c2.TeamID = t2.ID
	c2.SetTag("name", "U2 private collection")
	if err := s.CreateCollection(u2.ID, c2); err != nil {
		t.Fatal("Unable to create collection #2 in test environment: ", err)
	}

	c12 := model.NewCollection()
	c12.ID = s.NewCollectionID()
	c12.TeamID = t12.ID
	c12.SetTag("name", "Collection 1-2")
	if err := s.CreateCollection(u1.ID, c12); err != nil {
		t.Fatal("Unable to create collection #3 in test environment: ", err)
	}

	c21 := model.NewCollection()
	c21.ID = s.NewCollectionID()
	c21.TeamID = t21.ID
	c21.SetTag("name", "Collection 2-1")
	if err := s.CreateCollection(u2.ID, c21); err != nil {
		t.Fatal("Unable to create collection #4 in test environment: ", err)
	}

	c3 := model.NewCollection()
	c3.ID = s.NewCollectionID()
	c3.TeamID = t3.ID
	c3.SetTag("name", "The other collection")
	if err := s.CreateCollection(u3.ID, c3); err != nil {
		t.Fatal("Unable to create collection #5 in test environment: ", err)
	}

	return TestEnvironment{
		U1: u1, U2: u2, U3: u3,
		T1: t1, T2: t2, T12: t12, T21: t21, T3: t3,
		C1: c1, C2: c2, C12: c12, C21: c21, C3: c3}
}
