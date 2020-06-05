package restapi

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
	"database/sql"
	"strings"

	"github.com/ExploratoryEngineering/logging"
	"github.com/TelenorDigital/goconnect"
)

// ACMEParameters contains the autocert parameters
type ACMEParameters struct {
	Enabled   bool   `param:"desc=Enable ACME Certificates (aka Let's Encrypt) for host;default=false"`
	Hosts     string `param:"desc=ACME host names;default=api.nbiot.engineering,api.nbiot.telenor.io"`
	SecretDir string `param:"desc=ACME secrets directory;default=/var/horde/autocert"`
}

//HostList returns the list of hosts
func (p *ACMEParameters) HostList() []string {
	return strings.Split(p.Hosts, ",")
}

// ConnectIDParameters holds the configuration for the CONNECT ID OAuth client
type ConnectIDParameters struct {
	Enabled           bool   `param:"desc=Enable CONNECT ID integration;default=false"`
	Host              string `param:"desc=Host name for CONNECT ID OAuth service;default=connect.staging.telenordigital.com"`
	ClientID          string `param:"desc=Client ID for OAuth service;default=telenordigital-connectexample-web"`
	LoginRedirectURI  string `param:"desc=Redirect URI for OAuth client;default=http://localhost:8080/connect/oauth2callback"`
	LogoutRedirectURI string `param:"desc=Redirect URI for OAuth client;default=http://localhost:8080/connect/logoutcallback"`
	Password          string `param:"desc=OAuth client password;default="`
	LoginTarget       string `param:"desc=Final redirect after login;default=/"`
	LogoutTarget      string `param:"desc=Final redirect after logout;default=/"`
	Emulate           bool   `param:"desc=Emulate CONNECT ID;default=false"`
	sessionStore      *sql.DB
}

// SetSessionStoreConfig configures the session store
func (c *ConnectIDParameters) SetSessionStoreConfig(driver, connectionString string) error {
	if strings.HasPrefix(connectionString, ":memory:") {
		c.sessionStore = nil
		return nil
	}
	db, err := sql.Open(driver, connectionString)
	if err != nil {
		logging.Error("Unable to create session store: %v", err)
		return err
	}
	c.sessionStore = db
	return nil
}

// ConnectConfig returns the complete CONNECT ID config based on the parameters
func (c *ConnectIDParameters) ConnectConfig() goconnect.ClientConfig {
	cc := goconnect.ClientConfig{
		Host:                      c.Host,
		Password:                  c.Password,
		ClientID:                  c.ClientID,
		LoginRedirectURI:          c.LoginRedirectURI,
		LogoutRedirectURI:         c.LogoutRedirectURI,
		LoginCompleteRedirectURI:  c.LoginTarget,
		LogoutCompleteRedirectURI: c.LogoutTarget,
	}
	return goconnect.NewDefaultConfig(cc)
}

// ServerParameters contains the configuration parameters for the HTTP server
type ServerParameters struct {
	Endpoint         string `param:"desc=Listen address for HTTP server;default=localhost:8080"`
	TLSKeyFile       string `param:"desc=TLS key file;file"`
	TLSCertFile      string `param:"desc=TLS certificate file;file"`
	ACME             ACMEParameters
	RequestLog       string `param:"desc=Request log file name;default=requestlog.log"`
	InlineRequestlog bool   `param:"desc=Include request log in application log;default=false"`
}
