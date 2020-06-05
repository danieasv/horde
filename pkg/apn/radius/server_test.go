package radius

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
	"log"
	"net"
	"testing"

	rad "layeh.com/radius"
	"layeh.com/radius/rfc2865"
)

var testConfig = ServerParameters{
	Endpoint:     "127.0.0.1:0",
	SharedSecret: "radiussecret",
}

func TestServer(t *testing.T) {
	const goAwayMsg = "Go away"

	server := NewRADIUSServer(testConfig, func(r AccessRequest) AccessResponse {
		if r.Username == "bobok" {
			return AccessResponse{
				Accept:    true,
				IPAddress: net.ParseIP("10.0.0.1"),
			}
		}
		return AccessResponse{
			Accept:        false,
			RejectMessage: goAwayMsg,
		}
	})
	if err := server.Start(); err != nil {
		t.Fatal("Unable to start server: ", err)
	}
	defer func() {
		err := server.Stop()
		if err != nil {
			t.Fatal("Unable to stop server: ", err)
		}
	}()
	// Positive response
	{
		packet := rad.New(rad.CodeAccessRequest, []byte(testConfig.SharedSecret))
		rfc2865.UserName_SetString(packet, "bobok")
		rfc2865.UserPassword_SetString(packet, "test")

		response, err := rad.Exchange(context.Background(), packet, server.Address())
		if err != nil {
			log.Fatal("Error sending RADIUS request: ", err)
		}

		if response.Code != rad.CodeAccessAccept {
			t.Fatal("Client was rejected")
		}
	}

	// Negative response
	{
		// Set an AccessRequestHandler that will let anyone in and give
		// them the same IP address.
		packet := rad.New(rad.CodeAccessRequest, []byte(testConfig.SharedSecret))
		rfc2865.UserName_SetString(packet, "bobnotok")
		rfc2865.UserPassword_SetString(packet, "test")

		response, err := rad.Exchange(context.Background(), packet, server.Address())
		if err != nil {
			log.Fatal("Error sending RADIUS request: ", err)
		}

		if response.Code != rad.CodeAccessReject {
			t.Fatal("Client was NOT rejecteded")
		}

		msg := rfc2865.ReplyMessage_GetString(response)
		if msg != goAwayMsg {
			t.Fatalf("Reject message did not match, expected '%s' got '%s'", goAwayMsg, msg)
		}

	}
}
func TestWithCannedRequest(t *testing.T) {
	server := NewRADIUSServer(testConfig, func(r AccessRequest) AccessResponse {
		t.Logf("Got request %+v", r)
		const expectedIMEISV = "3541870799159337"
		if r.IMEISV != expectedIMEISV {
			return AccessResponse{
				Accept:        false,
				RejectMessage: "Wrong IMSI",
			}
		}

		const expectedIMSI = "242013057470867"
		if r.IMSI != expectedIMSI {
			return AccessResponse{
				Accept:        false,
				RejectMessage: "Wrong IMEI",
			}
		}

		return AccessResponse{
			Accept:    true,
			IPAddress: net.ParseIP("10.0.0.1"),
		}
	})
	if err := server.Start(); err != nil {
		t.Fatal("Unable to start server: ", err)
	}
	defer func() {
		err := server.Stop()
		if err != nil {
			t.Fatal("Unable to stop server: ", err)
		}
	}()

	packet := packetFromSample(t)

	response, err := rad.Exchange(context.Background(), packet, server.Address())
	if err != nil {
		log.Fatal("Error sending RADIUS request: ", err)
	}

	if response.Code != rad.CodeAccessAccept {
		t.Fatal("Client was rejected")
	}
}
