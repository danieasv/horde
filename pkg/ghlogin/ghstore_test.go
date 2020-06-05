package ghlogin
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
	"time"
)

func sessionStoreTest(ss SessionStore, t *testing.T) {
	// Get and set states
	// -----------------------------------------------------------------------
	if err := ss.PutState("state1"); err != nil {
		t.Fatal(err)
	}
	if err := ss.PutState("state1"); err == nil {
		t.Fatal("Should not be able to create the same state twice")
	}
	if err := ss.PutState("state2"); err != nil {
		t.Fatal(err)
	}

	if err := ss.RemoveState("state3"); err == nil {
		t.Fatal("Invalid state should yield error")
	}
	if err := ss.RemoveState("state2"); err != nil {
		t.Fatal(err)
	}
	if err := ss.RemoveState("state2"); err == nil {
		t.Fatal("Should not be able to remove state twice")
	}

	// Sessions
	// -----------------------------------------------------------------------

	if err := ss.CreateSession("1", "b", 1, Profile{Login: "u1"}); err != nil {
		t.Fatal(err)
	}
	if err := ss.CreateSession("2", "b", 1, Profile{Login: "u2"}); err != nil {
		t.Fatal(err)
	}

	if err := ss.CreateSession("1", "c", 1, Profile{Login: "u3"}); err == nil {
		t.Fatal("Should have unique session IDs")
	}

	prof, err := ss.GetSession("1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if prof.Profile.Login != "u1" {
		t.Fatal("Session 1 should be u1 but it was: ", prof)
	}
	if _, err := ss.GetSession("1", 999); err == nil {
		t.Fatal("Should not retrieve expired sessions")
	}

	if prof, err = ss.GetSession("2", 0); err != nil {
		t.Fatal(err)
	}
	if prof.Profile.Login != "u2" {
		t.Fatal("Session 2 should be u2 but it was: ", prof)
	}

	if _, err := ss.GetSession("9999", 999); err == nil {
		t.Fatal("Unknown session should be... unknown")
	}

	// neither should fail
	ss.RefreshSession("1", 1)
	ss.RefreshSession("2", 1)
	ss.RefreshSession("4", 1)
	ss.RefreshSession("99", 1)

	time.Sleep(100 * time.Nanosecond)
	// Check forward in time
	profiles, err := ss.GetSessions(time.Now().UnixNano())
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 2 {
		t.Fatal("Expected 2 elements but got ", len(profiles))
	}
	if err := ss.RemoveSession("1"); err != nil {
		t.Fatal(err)
	}
	if err := ss.RemoveSession("2"); err != nil {
		t.Fatal(err)
	}
	if err := ss.RemoveSession("1"); err == nil {
		t.Fatal("Session 1 should not exist")
	}
	if err := ss.RemoveSession("2"); err == nil {
		t.Fatal("Session 2 should not exist")
	}
}
