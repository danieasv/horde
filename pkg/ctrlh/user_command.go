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
	"context"
	"fmt"

	"github.com/eesrc/horde/pkg/managementproto"
)

// UserCommand is the subcommand for user management
type UserCommand struct {
	Add addUserCommand `kong:"cmd,help='Create new API user and associated token in Horde'"`
}

type addUserCommand struct {
	Name  string `kong:"help='User name for the new user',short='n'"`
	Email string `kong:"required,help='Email for the new user',short='e'"`
}

func (c *addUserCommand) Run(rc RunContext) error {
	service := connectToManagementServer(rc.HordeServer())
	if service == nil {
		return errStd
	}
	ctx, done := context.WithTimeout(context.Background(), grpcServerTimeout)
	defer done()

	resp, err := service.AddUser(ctx, &managementproto.AddUserRequest{
		Name:  rc.HordeCommands().User.Add.Name,
		Email: rc.HordeCommands().User.Add.Email,
	})
	if err != nil {
		fmt.Println("Could not add user: ", err)
		return err
	}
	if err := checkServiceResponse(resp.Result, err); err != nil {
		return err
	}

	fmt.Printf("User created.\nID = %s\nToken = %s\n", resp.UserId, resp.ApiToken)
	return nil
}
