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

// AuthMethod is an enum for authentication methods
type AuthMethod int

const (
	// AuthNone is "no auth method"
	AuthNone AuthMethod = iota

	// AuthGitHub is for GitHub authentication
	AuthGitHub

	// AuthConnectID is for CONNECT ID authentication
	AuthConnectID

	// AuthInternal is internal users. These are created through the management
	// API and have no external authentication. These users have a single API
	// token and nothing else.
	AuthInternal

	// AuthToken is token authentication
	AuthToken
)

// Login returns true if the user is authenticated via some sort of login, ie
// not via tokens or internal methods. Some API resources will need regular
// logins (like the token management bits, managing tokens via tokens introduces
// corner cases and potential security risks so we require a logged-in user to
// manage the tokens)
func (a AuthMethod) Login() bool {
	return a == AuthGitHub || a == AuthConnectID
}
