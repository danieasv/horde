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
	"time"

	"github.com/eesrc/horde/pkg/managementproto"
)

// AllocCommand is the subcommand for ip address allocations
type AllocCommand struct {
	APNID int                `kong:"required, help='APN ID',short='a'"`
	NASID int                `kong:"required, help='NAS ID',short='n'"`
	Add   allocAddCommand    `kong:"cmd,help='Add allocation for device'"`
	Rm    allocRemoveCommand `kong:"cmd,help='Remove allocation for device'"`
	List  allocListCommand   `kong:"cmd,help='List address allocations'"`
}

type allocAddCommand struct {
	IMSI      int64  `kong:"required,help='IMSI for device',short='i',default=''"`
	IPAddress string `kong:"required,help='IP for device',short='A',default=''"`
}

func (c *allocAddCommand) Run(rc RunContext) error {
	service := connectToManagementServer(rc.HordeServer())
	if service == nil {
		return errStd
	}
	ip := net.ParseIP(rc.HordeCommands().Alloc.Add.IPAddress)
	if ip == nil {
		fmt.Printf("Invalid IP for allocation\n")
		return errStd
	}

	ctx, done := context.WithTimeout(context.Background(), grpcServerTimeout)
	defer done()

	resp, err := service.AddAllocation(ctx, &managementproto.AddAllocationRequest{
		ApnID: int32(rc.HordeCommands().Alloc.APNID),
		NasID: int32(rc.HordeCommands().Alloc.NASID),
		IMSI:  rc.HordeCommands().Alloc.Add.IMSI,
		IP:    rc.HordeCommands().Alloc.Add.IPAddress,
	})
	if err != nil {
		fmt.Println("Could not add allocation: ", err)
		return err
	}
	if err := checkServiceResponse(resp.Result, err); err != nil {
		return err
	}
	fmt.Printf("Allocation created\n")

	return nil
}

type allocRemoveCommand struct {
	IMSI int64 `kong:"required,help='IMSI for device',short='i',default=''"`
}

func (c *allocRemoveCommand) Run(rc RunContext) error {
	service := connectToManagementServer(rc.HordeServer())
	if service == nil {
		return errStd
	}

	ctx, done := context.WithTimeout(context.Background(), grpcServerTimeout)
	defer done()

	resp, err := service.RemoveAPNAllocation(ctx, &managementproto.RemoveAPNAllocationRequest{
		ApnID: int32(rc.HordeCommands().Alloc.APNID),
		NasID: int32(rc.HordeCommands().Alloc.NASID),
		IMSI:  rc.HordeCommands().Alloc.Rm.IMSI,
	})
	if err != nil {
		fmt.Println("Could not remove allocation: ", err)
		return err
	}
	if err := checkServiceResponse(resp.Result, err); err != nil {
		return err
	}

	fmt.Printf("Allocation removed\n")
	return nil
}

type allocListCommand struct {
}

func (c *allocListCommand) Run(rc RunContext) error {
	service := connectToManagementServer(rc.HordeServer())
	if service == nil {
		return errStd
	}
	ctx, done := context.WithTimeout(context.Background(), grpcServerTimeout)
	defer done()

	resp, err := service.ListAPNAllocations(ctx, &managementproto.ListAPNAllocationsRequest{
		ApnID: int32(rc.HordeCommands().Alloc.APNID),
		NasID: int32(rc.HordeCommands().Alloc.NASID),
	})
	if err != nil {
		fmt.Println("Could not list allocations: ", err)
		return err
	}
	if err := checkServiceResponse(resp.Result, err); err != nil {
		return err
	}

	fmt.Printf("%-20s%-20s%-20s%s\n", "IMSI", "IMEI", "IP", "Created")
	for _, v := range resp.Allocations {
		created := time.Unix(v.Created, 0)
		fmt.Printf("%-20d%-20d%-20s%s\n", v.IMSI, v.IMEI, v.IP, created.Format(time.RFC3339))
	}
	return nil
}
