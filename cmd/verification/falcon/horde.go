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
	"math/rand"

	nbiot "github.com/telenordigital/nbiot-go"
)

var devices []nbiot.Device
var collection nbiot.Collection
var team nbiot.Team
var op nbiot.Output
var uop nbiot.Output

func removeDevices(client *nbiot.Client, coll nbiot.Collection, team nbiot.Team, devices []nbiot.Device) {
	for i := 0; i < len(devices); i++ {
		if err := client.DeleteDevice(coll.ID, devices[i].ID); err != nil {
			log.Printf("*** Error removing device %s: %v\n", devices[i].ID, err)
		}
	}
	if err := client.DeleteOutput(coll.ID, op.GetID()); err != nil {
		log.Printf("*** Error removing output %s/%s: %v", op.GetCollectionID(), op.GetID(), err)
	}
	if err := client.DeleteOutput(coll.ID, uop.GetID()); err != nil {
		log.Printf("*** Error removing output %s/%s: %v", uop.GetCollectionID(), uop.GetID(), err)
	}
	if err := client.DeleteCollection(coll.ID); err != nil {
		log.Printf("*** Error removing collection %s: %v", coll.ID, err)
	}
	if err := client.DeleteTeam(team.ID); err != nil {
		log.Printf("*** Error removing team %s: %v", team.ID, err)
	}
}

func createDevices(client *nbiot.Client, config args) bool {

	newTeam, err := client.CreateTeam(nbiot.Team{
		Tags: map[string]string{"Name": "Fakon team"},
	})
	if err != nil {
		log.Printf("Unable to create new team: %v\n", err)
		return false
	}
	team = newTeam
	coll, err := client.CreateCollection(nbiot.Collection{
		TeamID: newTeam.ID,
		Tags:   map[string]string{"Name": "Fakon collection"},
	})
	if err != nil {
		log.Printf("Unable to create collection for devices: %v", err)
		return false
	}
	collection = coll

	newOutput := nbiot.WebHookOutput{
		CollectionID: coll.ID,
		URL:          fmt.Sprintf("%swebhook", config.OutputWebhookEndpoint),
	}
	log.Printf("Attempted to create host with %s", newOutput.URL)
	tmop, err := client.CreateOutput(collection.ID, newOutput)
	if err != nil {
		log.Printf("Unable to create output: %v", err)
	}
	op = tmop

	newUDPOutput := nbiot.UDPOutput{
		CollectionID: coll.ID,
		Host:         config.UDPHost,
		Port:         4711,
	}
	tmpUDPop, err := client.CreateOutput(collection.ID, newUDPOutput)
	if err != nil {
		log.Printf("Unable to create UDP output: %v", err)
	}
	uop = tmpUDPop

	for i := 0; i < config.DeviceCount; i++ {
		imsi := rand.Int63n(999999999999999)
		imei := imsi
		device, err := client.CreateDevice(coll.ID, nbiot.Device{
			IMSI: fmt.Sprintf("%v", imei),
			IMEI: fmt.Sprintf("%v", imsi),
			Tags: map[string]string{
				"Name": fmt.Sprintf("Fakon device %d", i),
			},
		})
		if err != nil {
			log.Printf("Unable to create device: %v\n", err)
			return false
		}
		devices = append(devices, device)
		imsi++
		imei++
	}
	return true
}
