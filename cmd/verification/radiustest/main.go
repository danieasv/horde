package main

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
	"context"
	"errors"
	"fmt"
	"net"
	"os"

	"github.com/eesrc/horde/pkg/apn/radius/threegpp"

	"github.com/ExploratoryEngineering/params"
	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
)

type parameters struct {
	RADIUSEndpoint       string `param:"desc=Host:port string for RADIUS server;default=127.0.0.1:1812"`
	SharedSecret         string `param:"desc=Shared secret for the RADIUS server;default=secret"`
	ExpectedCIDR         string `param:"desc=Expected CIDR range;default=127.0.0.1/8"`
	AcceptExpected       bool   `param:"desc=Expect accept-response from RADIUS server;default=true"`
	AttrUserName         string `param:"desc=User name attribute in RADIUS request;default=4799887755"`
	AttrPassword         string `param:"desc=Password attribute in RADIUS request;default=password"`
	AttrNasIP            string `param:"desc=NAS IP Address attribute in RADIUS request;default=192.0.1.2"`
	AttrNasIdentifier    string `param:"desc=NAS identifier attribute in RADIUS request;default=NAS01"`
	AttrIMSI             string `param:"desc=IMSI attribute in RADIUS request;default=999912345678"`
	AttrMSTimeZone       string `param:"desc=MS Time Zone attribute in RADIUS request; default=@"`
	AttrUserLocationInfo string `param:"desc=User-location-info attribute in RADIUS request;default="`
	Failure              bool   `param:"desc=Expect error;default=false"`
}

func main() {
	var config parameters
	if err := params.NewEnvFlag(&config, os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}

	if err := verifyParameters(config); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(2)
	}

	fmt.Printf("Testing RADIUS server at %s...\n", config.RADIUSEndpoint)

	packet := radius.New(radius.CodeAccessRequest, []byte(config.SharedSecret))
	if config.AttrUserName != "" {
		rfc2865.UserName_SetString(packet, config.AttrUserName)
	}
	if config.AttrPassword != "" {
		rfc2865.UserPassword_AddString(packet, config.AttrPassword)
	}
	if config.AttrNasIP != "" {
		rfc2865.NASIPAddress_Add(packet, net.ParseIP(config.AttrNasIP))
	}
	if config.AttrNasIdentifier != "" {
		rfc2865.NASIdentifier_AddString(packet, config.AttrNasIdentifier)
	}
	if config.AttrIMSI != "" {
		threegpp.ThreeGPPIMSI_AddString(packet, config.AttrIMSI)
	}
	if config.AttrMSTimeZone != "" {
		threegpp.ThreeGPPMSTimeZone_AddString(packet, config.AttrMSTimeZone)
	}
	if config.AttrUserLocationInfo != "" {
		threegpp.ThreeGPPUserLocationInfo_AddString(packet, config.AttrUserLocationInfo)
	}

	resp, err := radius.Exchange(context.Background(), packet, config.RADIUSEndpoint)
	if err != nil {
		if config.Failure {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(3)
	}
	if config.Failure {
		fmt.Fprintf(os.Stderr, "Expected RADIUS call to fail but it didn't")
		os.Exit(3)
	}
	if config.AcceptExpected && resp.Code != radius.CodeAccessAccept {
		fmt.Fprintf(os.Stderr, "Did not receive Access-Accept from RADIUS server but got %s\n", resp.Code.String())
		os.Exit(4)
	}

	if resp.Code != radius.CodeAccessAccept {
		fmt.Fprintf(os.Stdout, "Response = %s\n", resp.Code.String())
		os.Exit(0)
	}

	_, network, err := net.ParseCIDR(config.ExpectedCIDR)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid exepected CIDR: %s\n", err.Error())
		os.Exit(2)
	}
	addr := rfc2865.FramedIPAddress_Get(resp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Server did not return an IP address")
		os.Exit(4)
	}
	if !network.Contains(addr) {
		fmt.Fprintf(os.Stderr, "Server returned IP %s but expected CIDR is %s\n", addr.String(), config.ExpectedCIDR)
		os.Exit(4)
	}
	fmt.Fprintf(os.Stdout, "Response = %s, IP = %s\n", resp.Code.String(), addr.String())
	os.Exit(0)
}

func verifyParameters(config parameters) error {
	if config.RADIUSEndpoint == "" {
		return errors.New("radius endpoint is empty")
	}
	if config.SharedSecret == "" {
		return errors.New("radius shared secret is empty")
	}
	if config.AttrNasIP != "" {
		if ip := net.ParseIP(config.AttrNasIP); ip == nil {
			return errors.New("nas ip address is invalid")
		}
	}
	if _, _, err := net.ParseCIDR(config.ExpectedCIDR); err != nil {
		return err
	}
	return nil
}
