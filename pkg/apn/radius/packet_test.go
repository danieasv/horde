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
	"log"
	"testing"

	rad "layeh.com/radius"
)

const capturedSecret = "radiussecret"

// SampleAccessRequest is a RADIUS Access-Request that closely
// resembles the ones we will encounter in the wild.
var sample = []byte{
	// Code 1: Access-Request
	0x01,

	// Id
	0x0,

	// Length
	0x01, 0x65,

	// Authenticator
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,

	// Calling station id
	0x1f,                   // Type 31
	0x0c,                   // Length = 12
	0x34, 0x37, 0x39, 0x37, // 4 7 9 7
	0x36, 0x35, 0x37, 0x30, // 6 5 7 0
	0x35, 0x33, // 5 3

	// User name
	0x01,                   // Type = 1
	0x0c,                   // Length = 12
	0x34, 0x37, 0x39, 0x37, // 4 7 9 7
	0x36, 0x35, 0x37, 0x30, // 6 5 7 0
	0x35, 0x33, // 5 3

	// NAS-IP-Address
	0x04,                   // Type = 4
	0x06,                   // Length = 6
	0x4d, 0x10, 0x01, 0xe0, // 77 16 1 224

	// NAS-Identifier
	0x20,                         // Type = 32
	0x07,                         // Length = 7
	0x54, 0x4e, 0x41, 0x53, 0x31, // T N A S 1

	// Called-Station-ID
	0x1e,                   // Type = 30
	0x0e,                   // Length = 14
	0x74, 0x64, 0x74, 0x32, // t d t 2
	0x2e, 0x6d, 0x64, 0x61, // . m d a
	0x74, 0x65, 0x73, 0x74, // t e s t

	// Service-Type
	0x06,                   // Type = 6
	0x06,                   // Length = 6
	0x00, 0x00, 0x00, 0x02, // 0 0 0 2 : (RFC 2865, 5.6)

	// Framed-Protocol
	0x07,                   // Type = 7
	0x06,                   // Length = 6
	0x00, 0x00, 0x00, 0x07, // 0 0 0 7 : (GPRS_PDP_Context)

	// NAS-Port-Type
	0x3d,                   // Type = 61
	0x06,                   // Length = 6
	0x00, 0x00, 0x00, 0x12, // 0 0 0 18 : (Wireless other)

	// Vendor specific
	0x1a,                   // Type = 26
	0x17,                   // Length = 23
	0x00, 0x00, 0x28, 0xaf, // Vendor id = 10415 : (3gpp)
	0x01,                   // Vendor type = 1 (3GPP-IMSI
	0x11,                   // Vendor length = 17
	0x32, 0x34, 0x32, 0x30, // 2 4 2 0
	0x31, 0x33, 0x30, 0x35, // 1 3 0 5
	0x37, 0x34, 0x37, 0x30, // 7 4 7 0
	0x38, 0x36, 0x37, // 8 6 7

	// Vendor specific
	0x1a,                   // Type = 26
	0x0d,                   // Length = 13
	0x00, 0x00, 0x28, 0xaf, // Vendor id = 10415 : (3GPP)
	0x08,                         // Vendor type = 8 (3GPP-IMSI-Mcc-Mnc)
	0x07,                         // Vendor length = 7
	0x32, 0x34, 0x32, 0x30, 0x31, // 2 4 2 0 1

	// Vendor specific
	0x1a,                   // Type = 26
	0x09,                   // Length = 9
	0x00, 0x00, 0x28, 0xaf, // Vendor id = 10415 : (3GPP)
	0x0a, // Vendor Type = 10 (3GPP-NSAPI)
	0x03, // Vendor Length = 3
	0x36, // 6

	// Vendor specific
	0x1a,                   // Type = 26
	0x0c,                   // Length = 12
	0x00, 0x00, 0x28, 0xaf, // Vendor id = 10415 : (3GPP)
	0x02,                   // Vendor Type = 2 (3GPP-Charging-Id)
	0x06,                   // Vendor Length = 6
	0x1d, 0x1c, 0x72, 0x69, // integer 488403561

	// Vendor specific
	0x1a,                   // Type = 26
	0x0c,                   // Length = 12
	0x00, 0x00, 0x28, 0xaf, // Vendor id = 10415 : (3GPP)
	0x0d,                   // Vendor type = 13 (3GPP-Charging-Characteristics)
	0x06,                   // Vendor length = 6
	0x30, 0x31, 0x30, 0x30, // 0100

	// Vendor specific
	0x1a,                   // Type = 26
	0x0c,                   // Length = 12
	0x00, 0x00, 0x28, 0xaf, // Vendor id = 10415 : (3GPP)
	0x06,                   // Vendor type = 6 (3GPP-SGSN-Address)
	0x06,                   // Vendor length = 6
	0xd9, 0x94, 0x90, 0x49, // 217.148.144.73

	// Vendor specific
	0x1a,                   // Type = 26
	0x0c,                   // Length = 12
	0x00, 0x00, 0x28, 0xaf, // Vendor id = 10415 : (3GPP)
	0x07,                   // Vendor type = 7 (3GPP-GGSN-Address)
	0x06,                   // Vendor length = 6
	0xd9, 0x94, 0x90, 0x4E, // (217.148.144.78)

	// Vendor specific
	0x1a,                   // Type = 26
	0x09,                   // Length = 9
	0x00, 0x00, 0x28, 0xaf, // Vendor id = 10415 : (3GPP)
	0x15, // Vendor type = 21 (3GPP-RAT-Type)
	0x03, // Vendor length = 3
	0x06, // ...

	// Vendor specific
	0x1a,                   // Type = 26
	0x18,                   // Length = 24
	0x00, 0x00, 0x28, 0xaf, // Vendor id = 10415 : (3GPP)
	0x14,                                           // Vendor type = 20 (3GPP-IMEISV)
	0x12,                                           // Vendor length = 18
	0x33, 0x35, 0x34, 0x31, 0x38, 0x37, 0x30, 0x37, // 35418707
	0x39, 0x39, 0x31, 0x35, 0x39, 0x33, 0x33, 0x37, // 99159337

	// Vendor specific
	0x1a,                   // Type = 26
	0x15,                   // Length = 21
	0x00, 0x00, 0x28, 0xaf, // Vendor id = 10415 : (3GPP)
	0x16, // Vendor type = 22 (3GPP-User-Location-Info)
	0x0f, // Vendor length = 15
	0x82, 0x42, 0xf2, 0x10,
	0x76, 0xc1, 0x42, 0xf2,
	0x10, 0x01, 0x02, 0xda, 0x04,

	// Problems after this

	// Vendor specific
	0x1a,                   // Type = 26
	0x0d,                   // Length = 13
	0x00, 0x00, 0x28, 0xaf, // Vendor id = 10415 : (3GPP)
	0x12,                         // Vendor type = 18 (3GPP-SGSN-Mcc-Mnc)
	0x07,                         // Vendor length = 7
	0x32, 0x34, 0x32, 0x30, 0x31, // 24201

	// Vendor specific
	0x1a,                   // Type = 26
	0x0d,                   // Length = 13
	0x00, 0x00, 0x28, 0xaf, // Vendor id = 10415 : (3GPP)
	0x09,                         // Vendor type = 9 (3GPP-GGSN-Mcc-Mnc)
	0x07,                         // Vendor length = 7
	0x32, 0x34, 0x32, 0x30, 0x31, // "24201"

	// Vendor specific
	0x1a,                   // Type = 26
	0x09,                   // Length = 9
	0x00, 0x00, 0x28, 0xaf, // Vendor id = 10415 : (3GPP)
	0x0c, // Vendor type = 12 (3GPP-Selection-Mode)
	0x03, // Vendor length = 3
	0x31, // "1"

	// Vendor specific
	0x1a,                   // Type = 26
	0x0a,                   // Length = 10
	0x00, 0x00, 0x28, 0xaf, // Vendor id = 10415 : (3GPP)
	0x17, // Vendor type = 23 (3GPP-MS-TimeZone)
	0x04, // Vendor length = 4
	0x40, 0x00,

	// Vendor specific
	0x1a,                   // Type = 26
	0x1f,                   // Length = 31
	0x00, 0x00, 0x28, 0xaf, // Vendor id = 10415 : (3GPP)
	0x05,                                           // Vendor type = 5 (3GPP-Negotiated-QoS-Profile)
	0x19,                                           // Vendor length = 25
	0x30, 0x38, 0x2D, 0x35, 0x38, 0x30, 0x39, 0x30, // "08-58090"
	0x30, 0x30, 0x37, 0x41, 0x31, 0x32, 0x30, 0x30, // "007A1200"
	0x30, 0x30, 0x37, 0x41, 0x31, 0x32, 0x30, // "007A120"

	// Vendor specific
	0x1a,                   // Type = 26
	0x0c,                   // Length = 12
	0x00, 0x00, 0x28, 0xaf, // Vendor id = 10415 : (3GPP)
	0x03,                   // Vendor type = 3 (3GPP-PDP-Type)
	0x06,                   // Vendor length = 6
	0x00, 0x00, 0x00, 0x00, // (ipv4)

	// Vendor specific
	0x1a,                   // Type = 26
	0x09,                   // Length = 9
	0x00, 0x00, 0x28, 0xaf, // Vendor id = 10415 : (3GPP)
	0x1a, // Vendor type = 26 (3GPP-Negotiated-DSCP)
	0x03, // Vendor length = 3
	0x00, // Value

	// NAS-Port
	0x05,                   // Type = 5
	0x06,                   // Length = 6
	0x00, 0x10, 0x72, 0x6a, // (1077866)

	// User-Password
	0x02,                   // Type = 2
	0x12,                   // Length = 18
	0x68, 0x65, 0x6d, 0x6d, // "hemmelig12345678"
	0x65, 0x6c, 0x69, 0x67,
	0x31, 0x32, 0x33, 0x34,
	0x35, 0x36, 0x37, 0x38,
}

// PacketFromSample returns a sample Access-Request RADIUS Packet object.
func packetFromSample(t *testing.T) *rad.Packet {
	//
	p, err := rad.Parse(sample, []byte(capturedSecret))
	if err != nil {
		log.Fatal("Failed to parse packet: ", err)
	}
	return p
}
