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
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/ExploratoryEngineering/params"
	nbiot "github.com/telenordigital/nbiot-go"
)

type args struct {
	SourceCIDR            string        `param:"desc=Source CIDR for devices;default=10.10.10.10/8"`
	SourceAPN             string        `param:"desc=APN for RADIUS requests;default=mda.test"`
	DeviceCount           int           `param:"desc=Device count;default=10"`
	MessageCount          int           `param:"desc=Message count;default=10"`
	DataEP                string        `param:"desc=Horde data endpoint;default=127.0.0.1:31415"`
	RadiusEP              string        `param:"desc=RADIUS endpoint;default=127.0.0.1:1812"`
	MessageInterval       time.Duration `param:"desc=Message interval;default=1s"`
	APIToken              string        `param:"desc=API Token for HTTP requests"`
	RemoveAfterTest       bool          `param:"desc=Remove device, collection, team, output after test; default=true"`
	RADIUSSharedSecret    string        `param:"desc=RADIUS pre-shared secret;default=radiussharedsecret"`
	WebhookEndpoint       string        `param:"desc=Local webhook endpoint; default=127.0.0.1:8888"`
	OutputWebhookEndpoint string        `param:"desc=Address of webhook endpoint externally; default=http://127.0.0.1:8888/"`
	RestAPI               string        `param:"desc=REST API endpoint for Horde;default=http://localhost:8080"`
	UDPHost               string        `param:"desc=UDP Host name to use;default=127.0.0.1"`
}

func main() {
	var config args

	if err := params.NewEnvFlag(&config, os.Args[1:]); err != nil {
		fmt.Printf("%v\n", err)
		return
	}

	if config.RestAPI == "" || config.APIToken == "" {
		log.Printf("REST API endpoint and API token must be specified")
		os.Exit(1)

	}
	startWebserver(config.WebhookEndpoint)
	startUDPServer()
	time.Sleep(1 * time.Second)
	client, err := nbiot.NewWithAddr(config.RestAPI, config.APIToken)
	if err != nil {
		log.Printf("Unable to connect to the Horde API: %v", err)
		os.Exit(1)
	}
	if !createDevices(client, config) {
		os.Exit(2)
	}
	log.Printf("Created %d devices in collection %s for team %s", config.DeviceCount, collection.ID, team.ID)

	wg := sync.WaitGroup{}
	wg.Add(len(devices))
	for i := 0; i < len(devices); i++ {
		go func(device nbiot.Device) {
			emulateDevice(device, config)
			wg.Done()
		}(devices[i])
	}

	log.Printf("Waiting for devices to complete....")
	wg.Wait()

	log.Printf("Devices are done")

	// Wait for drain
	log.Printf("Waiting 3s for draining")
	time.Sleep(3 * time.Second)
	if config.RemoveAfterTest {
		removeDevices(client, collection, team, devices)
	}
}
