package main

import (
	"fmt"
	"os"
	"time"

	"github.com/dustin/go-coap"
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

func readCoAPPull(cfg parameters) {
	// The CoAP host might not be available right away. Try until it comes
	// online

	online := false
	count := 0

	var conn *coap.Conn
	var err error
	for !online {
		conn, err = coap.Dial("udp", cfg.UpstreamCOAP)
		if err == nil {
			online = true
			continue
		}
		time.Sleep(100 * time.Millisecond)
		count++
		if count > 600 {
			fmt.Printf("Could not dial CoAP server at %s for 60s. Exiting", cfg.UpstreamCOAP)
			os.Exit(1)
		}
	}
	id := uint16(1)

	for {
		msg := coap.Message{
			Type:      coap.Confirmable,
			Code:      coap.GET,
			MessageID: id,
			Token:     []byte{1, 2, 3, 4, 5, 6, 7, 8},
		}

		// The path parameter is ignored by Horde.
		msg.SetPath([]string{"something/or/other"})
		resp, err := conn.Send(msg)
		if err != nil {
			time.Sleep(10 * time.Second)
			continue
		}
		if resp.Code == coap.NotFound {
			fmt.Println("CoAP pull: NotFound")
			time.Sleep(10 * time.Second)
			continue
		}

		if resp.Code != coap.Content {
			fmt.Printf("Expected Content response but got %s\n", resp.Code.String())
			time.Sleep(10 * time.Second)
			continue
		}
		if len(resp.Payload) == 0 {
			time.Sleep(5 * time.Second)
			continue
		}
		fmt.Printf("CoAP pull got %d bytes\n", len(resp.Payload))
		incrementPull()
		id++
		// Wait until the exchange has completed. See #176. This ensures that
		// this CoAP exchange and the next one doesn't meed mid-flight.
		time.Sleep(500 * time.Millisecond)
	}
}
