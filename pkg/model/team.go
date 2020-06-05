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

// Member is a (single) member in a team
type Member struct {
	Role RoleID
	User User // For caching
}

// NewMember creates a new Member instance
func NewMember(user User, roleID RoleID) Member {
	return Member{
		Role: roleID,
		User: user,
	}
}

// TeamKey is the ID for the team type
type TeamKey storageKey

// NewTeamKeyFromString parses a key as a string
func NewTeamKeyFromString(id string) (TeamKey, error) {
	k, err := newKeyFromString(id)
	return TeamKey(k), err
}

// String returns the string representation of the TeamKey
func (t TeamKey) String() string {
	return storageKey(t).String()
}

// The Team type is a group of users
type Team struct {
	ID      TeamKey
	Members []Member
	Tags
}

// NewTeam creates a new Team instace
func NewTeam() Team {
	return Team{Tags: NewTags()}
}

// IsMember returns true if the specified user ID is a member of the team. Not thread safe.
func (t *Team) IsMember(userID UserKey) bool {
	for _, v := range t.Members {
		if v.User.ID == userID {
			return true
		}
	}
	return false
}

// IsAdmin returns true if user is admin of team. Not thread safe
func (t *Team) IsAdmin(userID UserKey) bool {
	for _, v := range t.Members {
		if v.User.ID == userID && v.Role == AdminRole {
			return true
		}
	}
	return false
}

// AddMember adds member to team. Not thread safe
func (t *Team) AddMember(newMember Member) bool {
	if t.IsMember(newMember.User.ID) {
		return false
	}
	t.Members = append(t.Members, newMember)
	return true
}

// RemoveMember removes the user from the member list
func (t *Team) RemoveMember(userID UserKey) {
	for i, v := range t.Members {
		if v.User.ID == userID {
			t.Members = append(t.Members[0:i], t.Members[i+1:]...)
			return
		}
	}
}

// UpdateMember updates a member's role if the user exists
func (t *Team) UpdateMember(userID UserKey, roleID RoleID) {
	for i, v := range t.Members {
		if v.User.ID == userID {
			v.Role = roleID
			t.Members[i] = v
		}
	}
}

// GetMember returns the member with the specified user ID. An empty member
// entity is returned if the member doesn't exist.
func (t *Team) GetMember(userID UserKey) Member {
	for _, v := range t.Members {
		if v.User.ID == userID {
			return v
		}
	}
	return Member{}
}
