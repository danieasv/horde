package main

import (
	"fmt"

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

// Note: This will only download the first block from the firmware endpoint
func doDirectDownload(cfg parameters) error {
	conn, err := coap.Dial("udp", cfg.HordeEndpoint)
	if err != nil {
		return err
	}

	id := uint16(1)

	msg := coap.Message{
		Type:      coap.Confirmable,
		Code:      coap.GET,
		MessageID: id,
		Token:     []byte{1, 2, 3, 4, 5, 6, 7, 8},
	}

	msg.SetPath([]string{"fw"})
	resp, err := conn.Send(msg)
	if err != nil {
		fmt.Println("Error sending initial request: ", err)
		return err
	}
	if cfg.NoNew && resp.Code == coap.NotFound {
		fmt.Println("No firmware download and no expected")
		return nil
	}
	if cfg.NoNew {
		return fmt.Errorf("expected not found response but got %s", resp.Code.String())
	}
	expectedResponse := coap.Content
	if resp.Code != expectedResponse {
		return fmt.Errorf("expected %s response but got %s", expectedResponse.String(), resp.Code.String())
	}
	if len(resp.Payload) == 0 {
		return fmt.Errorf("zero bytes returned from service")
	}
	fmt.Println("Downloaded ", len(resp.Payload), "bytes")
	return nil
}
