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
	"encoding/binary"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/eesrc/horde/pkg/htest"
	nbiot "github.com/telenordigital/nbiot-go"
)

// Generate a payload for the device. Use the timestamp, IMSI and a sequence
// number
func generatePayload(seq int) []byte {
	ret := make([]byte, 12)
	binary.BigEndian.PutUint32(ret, uint32(seq))
	binary.BigEndian.PutUint64(ret[4:], uint64(time.Now().UnixNano()))
	return ret
}

// Emulate a device lifespan
func emulateDevice(device nbiot.Device, config args) {
	// Sleep a random inteval before sending the RADIUS request
	time.Sleep(time.Duration(rand.Int63n(int64(config.MessageInterval))))

	if !doRADIUSRequest(device, config) {
		log.Printf("RADIUS request for device with IMSI %v failed\n", device.IMSI)
		return
	}

	delay := rand.Int31n(200) + 50
	// RADIUS request OK, IP address is assigned.
	time.Sleep(time.Millisecond * time.Duration(delay))

	for i := 0; i < config.MessageCount; i++ {
		payload := generatePayload(i)
		dst := config.DataEP
		src := fmt.Sprintf("%s:%d", device.Tags["ip"], 12000)
		log.Printf("Sending %d bytes to %s from %s", len(payload), dst, src)
		if err := htest.SendUDPWithSource(dst, src, payload); err != nil {
			log.Printf("Error sending UDP: %v", err)
		}
		randomDelay := time.Duration(rand.Int31n(50))
		// Sleep the interval plus a random delay
		time.Sleep(config.MessageInterval + randomDelay)
	}

	log.Printf("Finished sending payload for device %v\n", device.IMSI)
}
