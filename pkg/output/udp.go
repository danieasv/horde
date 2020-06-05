package output

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
	"errors"
	"fmt"
	"net"
	"reflect"
	"sync"
	"time"

	"github.com/eesrc/horde/pkg/metrics"
	"github.com/eesrc/horde/pkg/output/outputconfig"
	"github.com/eesrc/horde/pkg/utils/audit"

	"github.com/ExploratoryEngineering/logging"
	"github.com/eesrc/horde/pkg/model"
)

type udp struct {
	status model.OutputStatus
	logs   Logger
	mutex  *sync.Mutex
}

// newUDP creates an UDP output.
func newUDP() Output {
	return &udp{
		mutex: &sync.Mutex{},
		logs:  NewLogger(),
	}
}

func init() {
	registerOutput("udp", newUDP)
}

func (u *udp) udpSender(messages <-chan interface{}, config model.OutputConfig) {
	if _, err := u.Validate(config); err != nil {
		u.logs.Append("Invalid config. Suspended output")
		return
	}
	host := config[outputconfig.UDPHost].(string)
	port := int(config[outputconfig.UDPPort].(float64))

	localAddr, err := net.ResolveUDPAddr("udp", ":0")
	if err != nil {
		u.logs.Append("Could not resolve local address")
		return
	}
	remoteAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		u.logs.Append("Could not resolve address")
		return
	}
	output, err := net.DialUDP("udp", localAddr, remoteAddr)
	if err != nil {
		u.logs.Append("Unable to dial UDP")
		return
	}
	for m := range messages {
		msg, ok := m.(model.DataMessage)
		if !ok {
			logging.Warning("Not a model.DataMessage on channel: %T", m)
			u.mutex.Lock()
			u.status.Received++
			u.status.ErrorCount++
			u.mutex.Unlock()
			continue
		}

		bytes, err := output.Write(msg.Payload)
		u.mutex.Lock()
		u.status.Received++
		if bytes != len(msg.Payload) {
			logging.Warning("Sent %d bytes, expected %d", bytes, len(msg.Payload))
		}
		if err != nil {
			u.logs.Append("Error sending payload")
			logging.Warning("Got error sending payload: %v", err)
			u.status.ErrorCount++
		} else {
			u.status.Forwarded++
			metrics.DefaultCoreCounters.MessagesForwardUDP.Add(1)
			audit.Log("UDP: Sent %d bytes to device with IMSI %d, Device ID=%s, Collection ID=%s, Target=%s",
				len(msg.Payload), msg.Device.IMSI,
				msg.Device.ID.String(), msg.Device.CollectionID.String(), remoteAddr.String())
		}
		u.mutex.Unlock()
	}
}

func (u *udp) Validate(config model.OutputConfig) (model.ErrorMessage, error) {
	// The port parameter is treated as float64 even though the config uses
	// integers; the JSON conversion returns all numbers as float64
	errs := validateConfig(config, []fieldSpec{
		fieldSpec{outputconfig.UDPHost, reflect.String, true},
		fieldSpec{outputconfig.UDPPort, reflect.Float64, true},
	})
	if len(errs) > 0 {
		return errs, errors.New("invalid config")
	}
	v1, ok1 := config[outputconfig.UDPHost]
	v2, ok2 := config[outputconfig.UDPPort]
	if ok1 && ok2 {
		host, okHost := v1.(string)
		port, okPort := v2.(float64)
		if okHost && okPort {
			check := newEndpointChecker(fmt.Sprintf("udp://%s:%d", host, int(port)))
			if !check.IsValidHost() {
				errs[outputconfig.UDPHost] = "Invalid host name"
			}
		}
	}

	return errs, nil
}

func (u *udp) Start(config model.OutputConfig, collectionFieldMask model.FieldMask, systemFieldMask model.FieldMask, messages <-chan interface{}) {
	if errs, err := u.Validate(config); err != nil {
		u.logs.Append("Invalid config. Output isn't started.")
		logging.Warning("Invalid config for output: %+v. Won't start", errs)
		return
	}
	go u.udpSender(messages, config)
}

func (u *udp) Stop(timeout time.Duration) {
	// this is closed automatically
}

func (u *udp) Logs() []model.OutputLogEntry {
	return u.logs.Entries()
}

func (u *udp) Status() model.OutputStatus {
	u.mutex.Lock()
	defer u.mutex.Unlock()
	ret := u.status
	return ret
}
