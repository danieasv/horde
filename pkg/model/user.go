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

// UserKey is the key type for users
type UserKey storageKey

// User is the struct representing system users.
type User struct {
	ID            UserKey // User ID
	Email         string  // Always set for GH users
	Phone         string  // Not used by GH
	Name          string  // Name of user
	ExternalID    string  // Identifier for CONNECT ID users
	Deleted       bool    // Deleted flag. Not used ATM
	VerifiedEmail bool    // Always true for GH users, possibly false for CONNECT ID users
	VerifiedPhone bool    // Always false for GH users
	AvatarURL     string  // Avatar URL (GH only; TD does not support avatars)
	PrivateTeamID TeamKey // The private team for the user
	AuthType      AuthMethod
}

// NewUser creates a new User instance
func NewUser(id UserKey, identifier string, authType AuthMethod, teamID TeamKey) User {
	return User{ID: id, ExternalID: identifier, PrivateTeamID: teamID, AuthType: authType}
}

// NewUserKeyFromString converts the string into a UserKey instance
func NewUserKeyFromString(id string) (UserKey, error) {
	k, err := newKeyFromString(id)
	return UserKey(k), err
}

func (u UserKey) String() string {
	return storageKey(u).String()
}
