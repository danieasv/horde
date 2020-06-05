package model
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
)

func TestTeam(t *testing.T) {
	t1 := NewTeam()
	user := NewUser(UserKey(1), "connectid", AuthConnectID, TeamKey(1))
	roleID := MemberRole
	m := NewMember(user, roleID)
	if !t1.AddMember(m) {
		t.Fatal("Couldn't add member to team")
	}
	if t1.AddMember(m) {
		t.Fatal("Should not be able to add member twice")
	}
	if !t1.IsMember(m.User.ID) {
		t.Fatal("Expected this to be a member")
	}
	if t1.IsAdmin(m.User.ID) {
		t.Fatal("Should not be admin")
	}
	t1.RemoveMember(user.ID)
	if t1.IsMember(m.User.ID) {
		t.Fatal("Should not longer be a member")
	}

	m.Role = AdminRole
	if !t1.AddMember(m) {
		t.Fatal("Couldn't add member")
	}
	m.Role = MemberRole
	if t1.AddMember(m) {
		t.Fatal("Should not be able to add member")
	}
	if !t1.IsAdmin(m.User.ID) {
		t.Fatal("Should be admin")
	}

	t1.UpdateMember(m.User.ID, MemberRole)

	mx := t1.GetMember(m.User.ID)
	if mx != m {
		t.Fatalf("GetMember doesn't return the correct member (expected %+v got %+v)", m, mx)
	}
	my := t1.GetMember(UserKey(9999))
	if my.User.ID != 0 {
		t.Fatalf("Should get empty member back")
	}
	t1.ID, _ = NewTeamKeyFromString("0")
	if _, err := NewTeamKeyFromString(t1.ID.String()); err != nil {
		t.Fatal("Expected valid conversion from String(): ", err)
	}
}
