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
	"net"

	"github.com/eesrc/horde/pkg/managementproto"
)

// NASCommand is the subcommand for NAS management
type NASCommand struct {
	Add  nasAddCommand    `cmd:"" help:"Add new NAS"`
	Rm   nasRemoveCommand `cmd:"" help:"Remove existing NAS"`
	List nasListCommand   `cmd:"" help:"List NASes"`
}

type nasAddCommand struct {
	APNID      int    `kong:"required,short='a',help='APN ID'"`
	NASID      int    `kong:"required,short='n',help='NAS ID'"`
	Identifier string `kong:"required,short='i',help='NAS identifier string'"`
	CIDR       string `kong:"required,short='c',help='CIDR for NAS'"`
}

func (c *nasAddCommand) Run(rc RunContext) error {
	service := connectToManagementServer(rc.HordeServer())
	if service == nil {
		return errStd
	}
	ctx, done := context.WithTimeout(context.Background(), grpcServerTimeout)
	defer done()

	if _, _, err := net.ParseCIDR(rc.HordeCommands().NAS.Add.CIDR); err != nil {
		fmt.Printf("Invalid CIDR range: %v\n", err)
		return errStd
	}

	resp, err := service.AddNAS(ctx, &managementproto.AddNASRequest{
		ApnID: int32(rc.HordeCommands().NAS.Add.APNID),
		NewRange: &managementproto.NASRange{
			NasID:         int32(rc.HordeCommands().NAS.Add.NASID),
			NasIdentifier: rc.HordeCommands().NAS.Add.Identifier,
			CIDR:          rc.HordeCommands().NAS.Add.CIDR,
		},
	})
	if err != nil {
		fmt.Println("Could not add NAS: ", err)
		return err
	}
	if err := checkServiceResponse(resp.Result, err); err != nil {
		return err
	}

	fmt.Printf("NAS range created successfully\n")
	return nil
}

type nasRemoveCommand struct {
	APNID int `kong:"required,short='a',help='APN ID'"`
	NASID int `kong:"required,short='n',help='NAS ID'"`
}

func (c *nasRemoveCommand) Run(rc RunContext) error {
	service := connectToManagementServer(rc.HordeServer())
	if service == nil {
		return errStd
	}
	ctx, done := context.WithTimeout(context.Background(), grpcServerTimeout)
	defer done()

	resp, err := service.RemoveNAS(ctx, &managementproto.RemoveNASRequest{
		ApnID: int32(rc.HordeCommands().NAS.Rm.APNID),
		NasID: int32(rc.HordeCommands().NAS.Rm.NASID),
	})
	if err != nil {
		fmt.Println("Could not remove NAS: ", err)
		return err
	}
	if err := checkServiceResponse(resp.Result, err); err != nil {
		return err
	}

	fmt.Printf("NAS range removed\n")
	return nil
}

type nasListCommand struct {
	APNID int `kong:"required,short='a',help='APN ID'"`
}

func (c *nasListCommand) Run(rc RunContext) error {
	service := connectToManagementServer(rc.HordeServer())
	if service == nil {
		return errStd
	}
	ctx, done := context.WithTimeout(context.Background(), grpcServerTimeout)
	defer done()

	resp, err := service.ListAPN(ctx, &managementproto.ListAPNRequest{})
	if err != nil {
		fmt.Println("Could not list NAS: ", err)
		return err
	}
	if err := checkServiceResponse(resp.Result, err); err != nil {
		return err
	}

	for _, v := range resp.APNs {
		if v.APN.ApnID == int32(rc.HordeCommands().NAS.List.APNID) {
			fmt.Printf("%-10s%-10s%s\n", "ID", "NAS ID", "CIDR")
			for _, nas := range v.NasRanges {
				fmt.Printf("%-10d%-10s%s\n", nas.NasID, nas.NasIdentifier, nas.CIDR)
			}
			return nil
		}
	}
	fmt.Printf("No NAS ranges found\n")
	return nil
}
