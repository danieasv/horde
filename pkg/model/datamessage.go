package model

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
	"strings"
	"time"
)

// Note that iota might be tempting here but the transport is persisted so
// the order of the transports should not matter.
const (
	// UDPTransport is upstream UDP
	UDPTransport = MessageTransport(0)
	// CoAPTransport is uses CaAP and is sent unsolicited to the device, ie
	// it's not part of a GET request from the device.
	CoAPTransport = MessageTransport(1)
	// CoAPPullTransport is CoAP but the client pulls the message. This is
	// is useful if you want a device to retrieve the CoAP messages itself
	// f.e. when there's power saving modes running or you can't run a full
	// blown CoAP server on the device.
	CoAPPullTransport = MessageTransport(2)
	// UDPPullTransport sends the UDP message in response when the client has
	// sent an upstream message. This can be useful if the device is in some
	// kind of power save mode and you want to schedule a message for later
	// There is no guarantee *when* the message will be delivered
	UDPPullTransport = MessageTransport(3)

	UnknownTransport = MessageTransport(999)
)

// String returns the string representation of the transport
func (m MessageTransport) String() string {
	switch m {
	case UDPTransport:
		return "udp"
	case CoAPPullTransport:
		return "coap-pull"
	case CoAPTransport:
		return "coap-push"
	case UDPPullTransport:
		return "udp-pull"
	default:
		return fmt.Sprintf("unknown transport(%d)", m)
	}
}

// MessageTransportFromString returns the MessageTransport type that matches.
// If nothing matches the unknown transport type is returned
func MessageTransportFromString(s string) MessageTransport {
	str := strings.ToLower(s)
	for i := MessageTransport(0); i < 4; i++ {
		if str == i.String() {
			return i
		}
	}
	// This is for backward compatability
	if s == "coap" {
		return CoAPTransport
	}
	return UnknownTransport
}

// UDPMetaData holds metadata for messages received via UDP
type UDPMetaData struct {
	LocalPort  int
	RemotePort int
}

// CoAPMetaData holds metadata for messages received on the CoAP interface
type CoAPMetaData struct {
	Code string
	Path string
}

// MessageTransport is the transport for (upstream) messages
type MessageTransport int

// DataMessage is the type for messages passed to and from devices. Final
// content is TBD.
type DataMessage struct {
	Device    Device
	Received  time.Time
	Payload   []byte
	Transport MessageTransport
	UDP       UDPMetaData
	CoAP      CoAPMetaData
}

// NewDataMessage creates a new DataMessage instance.
func NewDataMessage(device Device, payload []byte, transport MessageTransport, udp UDPMetaData, coap CoAPMetaData) DataMessage {
	return DataMessage{
		Device:    device,
		Payload:   payload,
		Received:  time.Now(),
		Transport: transport,
		UDP:       udp,
		CoAP:      coap,
	}
}
