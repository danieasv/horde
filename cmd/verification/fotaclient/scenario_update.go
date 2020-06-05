package main

//
// Copyright 2020 Telenor Digital AS
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
import (
	"fmt"
	"os"

	"github.com/dustin/go-coap"
	"github.com/eesrc/horde/pkg/fota/lwm2m"
	"github.com/eesrc/horde/pkg/fota/lwm2m/objects"
)

type updateRequired struct {
	deviceInfo objects.DeviceInformation
	client     *LwM2MClient
}

func (u *updateRequired) HandleRequest(msg *coap.Message) (*coap.Message, bool, error) {
	switch "/" + msg.PathString() {
	case lwm2m.FirmwareStatePath:
		ret := reportState(objects.Idle, msg)
		return ret, false, nil

	case lwm2m.DeviceInformationPath:
		return deviceInformationResponse(msg, u.deviceInfo), false, nil

	case lwm2m.FirmwareImageURIPath:
		return firmwareImageURI(msg), false, nil

	case lwm2m.FirmwareUpdatePath:
		return firmwareUpdatePath(msg), true, nil

	default:
		fmt.Println("Don't know how to handle path ", msg.PathString())
		os.Exit(4)
	}
	return nil, true, fmt.Errorf("don't know how to process message to %s", msg.PathString())
}
