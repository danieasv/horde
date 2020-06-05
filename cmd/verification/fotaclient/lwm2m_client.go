package main

import (
	"errors"
	"fmt"
	"math/rand"
	"sync/atomic"

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

// LwM2MTLVContent is the media type for "application/vnd.oma.lwm2m+tlv"
const LwM2MTLVContent = uint32(11542)

// Since there's absolutely *zero* lwm2m clients in go that works I'll have to roll my own. Crap.

// LwM2MClient is a simple client that pretends to do firmware updates. It only supports a few
// of the device information properties and the firmware object and resources.
type LwM2MClient struct {
	// Endpoint is the local endpoint
	Endpoint string
	// ServerEndpoint is the remote endpoint
	ServerEndpoint string

	// ClientID is the client ID reported to the LwM2M server
	ClientID string

	// SessionTimeout is the timeout (in seconds) for the LwM2M sessions
	SessionTimeout int

	// LwM2MVersion is the version reported during the registration. We won't
	// check it (this is a real world implementation) so it can be set to  just
	// about anything.
	LwM2MVersion string

	// Binding is the protocol binding reported to the LwM2M server. The default
	// is 'u' (UDP), 's' (SMS), 't' (TCP) and 'n' (non-IP). SMS and "non-IP"
	// (which is IP when you look at the implementation. I doubt they'll use
	// token ring or ethernet) so in reality it's 't' or 'u'. We don't care
	// about TCP yet so set it to just 'u'
	Binding string

	// Device properties for the /3/0 object
	DeviceProperties objects.DeviceInformation

	Connection *coap.Conn

	m *int32
}

// DialAndRun dials the server, then runs the scenario
func (l *LwM2MClient) DialAndRun(scenario ClientScenario) error {
	var err error
	l.Connection, err = coap.Dial("udp", l.ServerEndpoint)
	l.m = new(int32)
	atomic.StoreInt32(l.m, 1)
	if err != nil {
		return err
	}

	if scenario == nil {
		return errors.New("no scenario provided")
	}
	if err := l.register(); err != nil {
		return err
	}
	for {
		msg, err := l.Connection.Receive()
		if err != nil {
			continue
		}
		resp, exit, err := scenario.HandleRequest(msg)
		if err != nil {
			return err
		}
		if resp != nil {
			if _, err := l.Connection.Send(*resp); err != nil {
				return fmt.Errorf("error sending firmware state: %v", err)
			}
		}
		if exit {
			return nil
		}
	}
}

func (l *LwM2MClient) msgID() uint16 {
	return uint16(atomic.AddInt32(l.m, 1))
}

func (l *LwM2MClient) token() []byte {
	ret := make([]byte, 4)
	rand.Read(ret)
	return ret
}

// Register registers the client on a remote server
func (l *LwM2MClient) register() error {
	msg := coap.Message{}
	msg.MessageID = l.msgID()
	msg.Token = l.token()
	msg.Type = coap.Confirmable
	msg.Code = coap.POST
	msg.SetPath([]string{"rd"})
	msg.AddOption(coap.URIQuery, fmt.Sprintf("lt=%d", l.SessionTimeout))
	msg.AddOption(coap.URIQuery, fmt.Sprintf("lwm2m=%s", l.LwM2MVersion))
	msg.AddOption(coap.URIQuery, fmt.Sprintf("b=%s", l.Binding))
	msg.AddOption(coap.LocationQuery, fmt.Sprintf("id=%s", l.ClientID))

	_, err := l.Connection.Send(msg)
	return err
}
