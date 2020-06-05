package ctrlh

//
// Copyright 2020 Telenor Digital AS
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
import (
	"errors"
	"fmt"

	"github.com/eesrc/horde/pkg/managementproto"
)

// ManagementServerParameters is the configuration for the management server interface
type ManagementServerParameters struct {
	Endpoint         string `kong:"required,help='gRPC management endpoint',default='localhost:1234'"`
	TLS              bool   `kong:"help='TLS enabled for gRPC'"`
	CertFile         string `kong:"help='Client certificate for management service',type='path'"`
	HostnameOverride string `kong:"help='Host name override for certificate'"`
}

// CommandList is the commands for the management tool
type CommandList struct {
	Ping  PingCommand  `kong:"cmd,help='Ping the management service'"`
	APN   APNCommand   `kong:"cmd,help='APN subcommands'"`
	NAS   NASCommand   `kong:"cmd,help='NAS subcommands'"`
	Alloc AllocCommand `kong:"cmd,help='Device IP address allocations'"`
	Token TokenCommand `kong:"cmd,help='API token management'"`
	User  UserCommand  `kong:"cmd,help='User management'"`
	Util  UtilCommand  `kong:"cmd,help='Misc utiltiies'"`
}

// RunContext is the common context for the commands. The Command type
// implements this interface and can be used as is.
type RunContext interface {
	HordeServer() ManagementServerParameters
	HordeCommands() CommandList
}

// Command is the default configuration for the management tool
type Command struct {
	Server   ManagementServerParameters `kong:"embed"`
	Commands CommandList                `kong:"embed"`
}

// HordeServer returns the management server parameters
func (c *Command) HordeServer() ManagementServerParameters {
	return c.Server
}

// HordeCommands returns the mangement server command list
func (c *Command) HordeCommands() CommandList {
	return c.Commands
}

// this is a generic error returned by the commands since we're using the
// standard output for more informative error messages
var errStd = errors.New("error")

// helper function to check the service response. The error messages are
// identical for all requests.
func checkServiceResponse(resp *managementproto.Result, err error) error {
	if err != nil {
		fmt.Printf("Error calling management service: %v\n", err)
		return errStd
	}
	if !resp.Success {
		fmt.Printf("Service reported error: %s\n", resp.Error)
		return errStd
	}
	return nil
}
