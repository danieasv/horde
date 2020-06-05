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
	"errors"
	"fmt"

	"github.com/eesrc/horde/pkg/managementproto"
)

// APNCommand is the subcommand for APN management
type APNCommand struct {
	Add    apnAddCommand    `kong:"cmd,help='Add new APN in Horde'"`
	Rm     apnRemoveCommand `kong:"cmd,help='Remove APN from Horde'"`
	List   apnListCommand   `kong:"cmd,help='List APNs registered in Horde'"`
	Reload apnReloadCommand `kong:"cmd,help='Reload APN and NAS list'"`
}

type apnAddCommand struct {
	APNID int    `kong:"required,help='ID of new APN',short='a'"`
	Name  string `kong:"required,help='Name of new APN',short='n'"`
}

func (c *apnAddCommand) Run(rc RunContext) error {
	service := connectToManagementServer(rc.HordeServer())
	if service == nil {
		return errStd
	}

	ctx, done := context.WithTimeout(context.Background(), grpcServerTimeout)
	defer done()

	resp, err := service.AddAPN(ctx, &managementproto.AddAPNRequest{
		NewAPN: &managementproto.APN{
			ApnID: int32(rc.HordeCommands().APN.Add.APNID),
			Name:  rc.HordeCommands().APN.Add.Name,
		},
	})
	if err != nil {
		fmt.Println("Could not add APN: ", err)
		return err
	}
	if err := checkServiceResponse(resp.Result, err); err != nil {
		return err
	}

	fmt.Printf("APN created successfully\n")

	return nil
}

type apnRemoveCommand struct {
	APNID int `kong:"required,help='APN ID',short='a'"`
}

func (c *apnRemoveCommand) Run(rc RunContext) error {
	service := connectToManagementServer(rc.HordeServer())
	if service == nil {
		return errStd
	}

	ctx, done := context.WithTimeout(context.Background(), grpcServerTimeout)
	defer done()

	resp, err := service.RemoveAPN(ctx, &managementproto.RemoveAPNRequest{
		ApnID: int32(rc.HordeCommands().APN.Rm.APNID),
	})
	if err != nil {
		fmt.Println("Could not remove APN: ", err)
		return err
	}
	if err := checkServiceResponse(resp.Result, err); err != nil {
		return err
	}

	fmt.Printf("APN removed\n")
	return nil
}

type apnListCommand struct {
}

func (c *apnListCommand) Run(rc RunContext) error {
	service := connectToManagementServer(rc.HordeServer())
	if service == nil {
		return errors.New("no service")
	}
	ctx, done := context.WithTimeout(context.Background(), grpcServerTimeout)
	defer done()

	resp, err := service.ListAPN(ctx, &managementproto.ListAPNRequest{})
	if err != nil {
		fmt.Println("Could not list APN: ", err)
		return err
	}
	if err := checkServiceResponse(resp.Result, err); err != nil {
		return err
	}

	fmt.Printf("%-10s%s\n", "ID", "Name")
	for _, v := range resp.APNs {
		fmt.Printf("%-10d%s\n", v.APN.ApnID, v.APN.Name)
	}
	return nil
}

type apnReloadCommand struct {
}

func (c *apnReloadCommand) Run(rc RunContext) error {
	service := connectToManagementServer(rc.HordeServer())
	if service == nil {
		return errors.New("no service")
	}
	ctx, done := context.WithTimeout(context.Background(), grpcServerTimeout)
	defer done()

	resp, err := service.ReloadAPN(ctx, &managementproto.ReloadAPNRequest{})
	if err != nil {
		fmt.Println("Could not reload APN: ", err)
		return err
	}
	if err := checkServiceResponse(resp.Result, err); err != nil {
		return err
	}
	fmt.Println("Reloaded APN")
	return nil
}
