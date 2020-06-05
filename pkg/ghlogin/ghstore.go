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

// Session is the stored session data
type Session struct {
	Expires     int64
	SessionID   string
	AccessToken string
	Profile     Profile
}

// SessionStore is the interface for the session back-end store.
type SessionStore interface {
	// PutState inserts a new state nonce in the storage. The nonce will never expire
	PutState(state string) error

	// RemoveState removes a state nonce from the storage. An error is returned if the
	// nonce does not exist.
	RemoveState(state string) error

	// CreateSession creates a new session in the store. The lastUpdate parameter is the expire
	// time (in ns) for the session
	CreateSession(sessionID string, accessToken string, expires int64, profile Profile) error

	// GetSession returns the session from. The ignoreOlder is the current time stamp
	GetSession(sessionID string, ingnoreOlder int64) (Session, error)

	// RemoveSession removes the session from the store
	RemoveSession(sessionID string) error

	// GetSessions returns sessions with the last_update parameter set to the current value
	GetSessions(time int64) ([]Session, error)

	// RefreshSession refreshes a session expire time
	RefreshSession(sessionID string, checkInterval int64) error
}
