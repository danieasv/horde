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

// TestTeamStore tests team store implementations. The users must be created by the caller.
func testTeamStore(env TestEnvironment, store storage.DataStore, t *testing.T) {
	id1 := store.NewTeamID()
	id2 := store.NewTeamID()
	if id1 == id2 {
		t.Fatal("Expected unique IDs but they were the same")
	}

	t1 := model.NewTeam()
	t1.ID = store.NewTeamID()
	t1.AddMember(model.NewMember(env.U1, model.AdminRole))
	t1.Tags.SetTag("name", "Team 1")
	if err := store.CreateTeam(t1); err != nil {
		t.Fatal("Couldn't create team 1:", err)
	}

	t1.AddMember(model.NewMember(env.U2, model.AdminRole))
	if err := store.UpdateTeam(env.U1.ID, t1); err != nil {
		t.Fatal("Unable to update team first time")
	}
	t1.RemoveMember(env.U2.ID)
	if err := store.UpdateTeam(env.U1.ID, t1); err != nil {
		t.Fatal("Unable to update team 2nd time")
	}
	t1.AddMember(model.NewMember(env.U2, model.AdminRole))
	store.UpdateTeam(env.U1.ID, t1)

	t1.RemoveMember(env.U2.ID)
	t1.AddMember(model.NewMember(env.U2, model.MemberRole))
	if err := store.UpdateTeam(env.U1.ID, t1); err != nil {
		t.Fatal("Unable to update team 3rd time")
	}

	t2 := model.NewTeam()
	t2.ID = store.NewTeamID()
	t2.AddMember(model.NewMember(env.U2, model.AdminRole))
	t2.AddMember(model.NewMember(env.U1, model.MemberRole))
	t2.Tags.SetTag("name", "Team 2")

	t3 := model.NewTeam()
	t3.ID = store.NewTeamID()
	t3.AddMember(model.NewMember(env.U1, model.AdminRole))
	t3.Tags.SetTag("name", "Team 3")

	// ------- Create
	if err := store.CreateTeam(t2); err != nil {
		t.Fatal("Couldn't create team 2: ", err)
	}
	if err := store.CreateTeam(t2); err != storage.ErrAlreadyExists {
		t.Fatal("Expected error but got ", err)
	}
	store.CreateTeam(t3)

	isInList := func(teams []model.Team, teamIDs ...model.TeamKey) bool {
		if len(teams) < len(teamIDs) {
			return false
		}
		found := 0
		for _, k := range teamIDs {
			for _, i := range teams {
				if k == i.ID {
					found++
				}
			}
		}
		return found == len(teamIDs)
	}
	// ------- List
	list, err := store.ListTeams(env.U1.ID)
	if err != nil {
		t.Fatal("Couldn't retrieve team list 1: ", err)
	}
	if !isInList(list, t1.ID, t2.ID) {
		t.Fatalf("Expected team 1 and 2 to be in list: %+v", list)
	}

	list, err = store.ListTeams(env.U2.ID)
	if err != nil {
		t.Fatal("Couldn't retrieve team list 2: ", err)
	}
	if !isInList(list, t1.ID, t2.ID) {
		t.Fatalf("Expected team 1 and 2 to be in list: %+v", list)
	}

	// ------- Retrieve
	if _, err := store.RetrieveTeam(env.U1.ID, t1.ID); err != nil {
		t.Fatal("Expected t1 to exist: ", err)
	}
	if _, err := store.RetrieveTeam(env.U1.ID, store.NewTeamID()); err != storage.ErrNotFound {
		t.Fatal("Expected not found error: ", err)
	}

	if err := store.UpdateTeam(env.U1.ID, t1); err != nil {
		t.Fatal("Should be able to update team 1: ", err)
	}
	if err := store.UpdateTeam(env.U2.ID, t2); err != nil {
		t.Fatal("Should be able to update team 2: ", err)
	}
	if err := store.UpdateTeam(env.U1.ID, env.T21); err != storage.ErrAccess {
		t.Fatal("Should not be able to update team I'm not an admin of. err = ", err)
	}
	if err := store.UpdateTeam(env.U1.ID, env.T3); err != storage.ErrNotFound {
		t.Fatal("Should not be able to update team I'm not a member of. err = ", err)
	}
	t1r, err := store.RetrieveTeam(env.U1.ID, t1.ID)
	if err != nil {
		t.Fatal("Unable to retrieve team 1: ", err)
	}
	t2r, err := store.RetrieveTeam(env.U1.ID, t2.ID)
	if err != nil {
		t.Fatal("Unable to retrieve team 2: ", err)
	}
	if !t1r.IsAdmin(env.U1.ID) {
		t.Fatal("user 1 should be a member of team 2")
	}
	if !t1r.IsMember(env.U2.ID) {
		t.Fatal("user 2 should be a member of team 1")
	}
	if !t2r.IsAdmin(env.U2.ID) {
		t.Fatal("user 2 should be admin of team 1")
	}
	if !t2r.IsMember(env.U1.ID) {
		t.Fatal("user 1 should be a member of team 2")
	}

	// Ensure tags are retrieved correctly
	if t1r.GetTag("name") != "Team 1" {
		t.Fatal("Team 1 did not get a name tag: ", t1r.TagMap)
	}
	if t2r.GetTag("name") != "Team 2" {
		t.Fatal("Team 2 did not get a name tag")
	}
	// ------- Tags
	store.UpdateTeam(env.U1.ID, t1)
	testTagSetAndGet(t, env, t1.ID.String(), true, store.UpdateTeamTags, store.RetrieveTeamTags)

	// ------ TeamDelete
	if err := store.DeleteTeam(env.U2.ID, t1.ID); err != storage.ErrAccess {
		t.Fatal("Non-admins shouldn't be able to remove teams but got : ", err)
	}
	t1.RemoveMember(env.U2.ID)
	store.UpdateTeam(env.U1.ID, t1)
	if err := store.DeleteTeam(env.U2.ID, t1.ID); err != storage.ErrNotFound {
		t.Fatal("Non-member shouldn't be able to remove teams but got : ", err)
	}
	if err := store.DeleteTeam(env.U2.ID, t2.ID); err != nil {
		t.Fatal("couldn't remove team 2: ", err)
	}
	if err := store.DeleteTeam(env.U1.ID, t1.ID); err != nil {
		t.Fatal("Couldn't remove team 1: ", err)
	}
	if err := store.DeleteTeam(env.U1.ID, t1.ID); err != storage.ErrNotFound {
		t.Fatal("Expected not found but got: ", err)
	}

	// Operations on non-existing teams
	if err := store.UpdateTeam(env.U2.ID, t2); err != storage.ErrNotFound {
		t.Fatal("Shouldn't be able to update nonexisting teams")
	}
}
