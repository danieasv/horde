package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/eesrc/horde/pkg/fota/lwm2m/objects"

	"github.com/ExploratoryEngineering/params"
)

//
//Copyright 2020 Telenor Digital AS
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

type parameters struct {
	Version         string `param:"desc=Reported version;default=1.0.0"`
	Manufacturer    string `param:"desc=Reported manufacturer;default=Exploratory Engineering"`
	Serial          string `param:"desc=Reported serial number;default=1"`
	Model           string `param:"desc=Reported model number;default=EE01"`
	HordeEndpoint   string `param:"desc=Horde endpoint;default=127.0.0.1:5683"`
	Scenario        string `param:"desc=Scenario to use;default=update;options=update,noupdate,error,nonidle"`
	DownloadTimeout int    `param:"desc=Timeout for downloads in Horde;default=45"`
	Direct          bool   `param:"desc=Attempt direct download of firmware;default=false"`
	Simple          bool   `param:"desc=Use simple FOTA process;default=false"`
	NoNew           bool   `param:"desc=No update expected (for direct download);default=false"`
}

func main() {
	rand.Seed(time.Now().UnixNano())

	cfg := parameters{}
	if err := params.NewEnvFlag(&cfg, os.Args[1:]); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	if cfg.Direct {
		if err := doDirectDownload(cfg); err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		os.Exit(0)
	}

	if cfg.Simple {
		if err := doSimpleFOTA(cfg); err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		os.Exit(0)
	}
	deviceInfo := objects.DeviceInformation{
		FirmwareVersion: cfg.Version,
		Manufacturer:    cfg.Manufacturer,
		SerialNumber:    cfg.Serial,
		ModelNumber:     cfg.Model,
		BatteryLevel:    100,
	}
	client := LwM2MClient{
		Endpoint:       ":0",
		ServerEndpoint: cfg.HordeEndpoint,
		ClientID:       fmt.Sprintf("%06x", rand.Int()),
		SessionTimeout: 300,
		LwM2MVersion:   "1",
		Binding:        "u",
	}

	var scenario ClientScenario

	switch cfg.Scenario {
	case "update":
		scenario = &updateRequired{deviceInfo: deviceInfo, client: &client}

	case "noupdate":
		scenario = &noUpdateRequired{deviceInfo: deviceInfo}

	case "error":
		scenario = &imageError{deviceInfo: deviceInfo, client: &client}

	case "nonidle":
		scenario = &nonIdle{deviceInfo: deviceInfo}

	default:
		fmt.Println("Don't know how to create scenario ", cfg.Scenario)
		os.Exit(2)
	}
	go func() {
		<-time.After(60 * time.Second)
		fmt.Println("Scenario timed out")
		os.Exit(3)
	}()

	if err := client.DialAndRun(scenario); err != nil {
		fmt.Println("Error running scenario: ", err)
		os.Exit(1)
	}
	os.Exit(0)
}
