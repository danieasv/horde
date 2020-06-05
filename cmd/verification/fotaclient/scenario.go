package main

import (
	"fmt"
	"net/url"
	"os"

	"github.com/dustin/go-coap"
	"github.com/eesrc/horde/pkg/fota/lwm2m/objects"
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

// ClientScenario is a single scenario for the fota client.
type ClientScenario interface {
	// HandleRequest handles a CoAP request from the server. The returned
	// message should be sent (if it is non-nil) and if the boolean value
	// is true the serve function returns
	HandleRequest(msg *coap.Message) (*coap.Message, bool, error)
}

func reportState(state objects.FirmwareUpdateState, msg *coap.Message) *coap.Message {
	respMsg := &coap.Message{
		Type:      coap.NonConfirmable,
		Code:      coap.Content,
		MessageID: msg.MessageID,
		Token:     msg.Token,
	}
	respMsg.SetOption(coap.LocationPath, msg.Path())
	respMsg.SetOption(coap.ContentFormat, LwM2MTLVContent)
	respMsg.Payload = objects.EncodeBytes(3, []byte{byte(state)})

	fmt.Printf("Returning state %s to server\n", state.String())
	return respMsg
}

func deviceInformationResponse(msg *coap.Message, info objects.DeviceInformation) *coap.Message {
	resp := coap.Message{}
	resp.MessageID = msg.MessageID
	resp.Token = msg.Token
	resp.Type = coap.NonConfirmable
	resp.Code = coap.Content
	resp.SetOption(coap.ContentFormat, LwM2MTLVContent)

	buf := info.Buffer()
	resp.Payload = buf
	return &resp
}

func firmwareImageURI(msg *coap.Message) *coap.Message {
	resp := coap.Message{}
	resp.MessageID = msg.MessageID
	resp.Token = msg.Token
	resp.Type = coap.NonConfirmable

	resp.Code = coap.Changed
	resp.SetPath(msg.Path())

	buf := objects.NewTLVBuffer(msg.Payload)

	if len(buf.Resources) == 0 {
		fmt.Printf("No firmware image path is sent to client")
		os.Exit(1)
	}
	if len(buf.Resources) > 0 {
		fmt.Printf("Firmware image URI path is set to %+v\n", buf.Resources[0].String())
		u, err := url.Parse(buf.Resources[0].String())
		if err != nil {
			fmt.Printf("Got error parsing URI: %v\n", err)
			os.Exit(1)
		}
		go func(endpoint string) {
			/*			// The download won't be imidiate since there's latency in the
						// network. Sleep for a second before continuing. Real latency will
						// be more like a few seconds but shorter is better.
						time.Sleep(100 * time.Millisecond)*/
			if err := doDirectDownload(parameters{
				HordeEndpoint: endpoint,
			}); err != nil {
				fmt.Printf("Error downloading: %v\n", err)
				os.Exit(1)
			}
		}(u.Host)
	}

	return &resp

}

func firmwareUpdatePath(msg *coap.Message) *coap.Message {
	if msg.Code == coap.POST {
		fmt.Println("Server requested update")
		resp := coap.Message{}
		resp.MessageID = msg.MessageID
		resp.Token = msg.Token
		resp.Type = coap.NonConfirmable
		resp.Code = coap.Valid
		resp.SetOption(coap.LocationPath, msg.Path())
		// State is updated. Restart the process and report the new version number.
		return &resp
	}
	return nil
}

func notFoundError(msg *coap.Message) *coap.Message {
	resp := coap.Message{}
	resp.MessageID = msg.MessageID
	resp.Token = msg.Token
	resp.Type = coap.NonConfirmable
	resp.Code = coap.NotFound
	resp.SetOption(coap.LocationPath, msg.Path())
	return &resp
}
