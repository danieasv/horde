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

// Config is the GitHub OAuth application settings. Create new one at
// https://github.com/settings/developers. You must enroll into the GH
// Developer Program to create an application. Read more at
// https://developer.github.com/
type Config struct {
	ClientID           string `param:"desc=OAuth client ID"`
	CallbackURL        string `param:"desc=Callback URL for client;default=http://localhost:8080/github/callback"`
	ClientSecret       string `param:"desc=OAuth Client secret"`
	LoginSuccess       string `param:"desc=Redirect after successful login;default=/"`
	LogoutSuccess      string `param:"desc=Redirect after logout;default=/"`
	SecureCookie       bool   `param:"desc=Secure flag on cookie;default=false"`
	DBDriver           string `param:"desc=Database driver (postgres, sqlite3);default=sqlite3"`
	DBConnectionString string `param:"desc=Connection string for session store;default=:memory:"`
}
