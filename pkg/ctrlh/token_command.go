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

// TokenCommand is the token subcommand
type TokenCommand struct {
	Add addTokenCommand    `kong:"cmd,help='Create a new API token for an user'"`
	Rm  removeTokenCommand `kong:"cmd,help='Remove an API token from an existing user'"`
}

type addTokenCommand struct {
	UserID string `kong:"required,help='User ID of user'"`
}

func (c *addTokenCommand) Run(rc RunContext) error {
	service := connectToManagementServer(rc.HordeServer())
	if service == nil {
		return errStd
	}
	ctx, done := context.WithTimeout(context.Background(), grpcServerTimeout)
	defer done()

	resp, err := service.AddToken(ctx, &managementproto.AddTokenRequest{
		UserId: rc.HordeCommands().Token.Add.UserID,
	})
	if err != nil {
		fmt.Println("Could not add token: ", err)
		return err
	}
	if err := checkServiceResponse(resp.Result, err); err != nil {
		return err
	}

	fmt.Printf("Token created for user %s\nToken: %s\n", rc.HordeCommands().Token.Add.UserID, resp.ApiToken)

	return nil
}

type removeTokenCommand struct {
	UserID string `kong:"required,help='User ID of user',short='u'"`
	Token  string `kong:"required,help='Token to remove',short='t'"`
}

func (c *removeTokenCommand) Run(rc RunContext) error {
	service := connectToManagementServer(rc.HordeServer())
	if service == nil {
		return errStd
	}
	ctx, done := context.WithTimeout(context.Background(), grpcServerTimeout)
	defer done()

	resp, err := service.RemoveToken(ctx, &managementproto.RemoveTokenRequest{
		UserId:   rc.HordeCommands().Token.Rm.UserID,
		ApiToken: rc.HordeCommands().Token.Rm.Token,
	})
	if err != nil {
		fmt.Println("Could not remove token: ", err)
		return err
	}
	if err := checkServiceResponse(resp.Result, err); err != nil {
		return err
	}

	fmt.Printf("Token %s removed for user %s\n", rc.HordeCommands().Token.Rm.Token, rc.HordeCommands().Token.Rm.UserID)

	return nil
}
