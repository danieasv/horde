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
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDataMessage(t *testing.T) {
	d := NewDevice()
	d.ID, _ = NewDeviceKeyFromString("1")
	m := NewDataMessage(NewDevice(), []byte{1, 2, 3, 4, 5}, UDPTransport, UDPMetaData{}, CoAPMetaData{})
	if m.Payload == nil {
		t.Fatal("Expected payload to be non-nil")
	}
}

func TestDataMessageString(t *testing.T) {
	assert := require.New(t)

	assert.Equal("coap-push", CoAPTransport.String())
	assert.Equal("coap-pull", CoAPPullTransport.String())
	assert.Equal("udp", UDPTransport.String())
	assert.Equal("udp-pull", UDPPullTransport.String())
	assert.Equal("unknown transport(12)", MessageTransport(12).String())

	assert.Equal(CoAPPullTransport, MessageTransportFromString("coap-pull"))
	assert.Equal(CoAPTransport, MessageTransportFromString("coap"))
	assert.Equal(CoAPTransport, MessageTransportFromString("coap-push"))
	assert.Equal(UDPTransport, MessageTransportFromString("udp"))
	assert.Equal(UDPPullTransport, MessageTransportFromString("udp-pull"))
	assert.Equal(UnknownTransport, MessageTransportFromString("unknown"))
}
