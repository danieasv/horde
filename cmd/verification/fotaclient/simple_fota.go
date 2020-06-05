package main

import (
	"errors"
	"fmt"

	"github.com/eesrc/horde/pkg/fota"
	"github.com/eesrc/horde/pkg/htest"

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

func doSimpleFOTA(config parameters) error {
	conn, err := coap.Dial("udp", config.HordeEndpoint)
	if err != nil {
		return err
	}

	// Encode request, then post it

	report := fota.Report{
		FirmwareVersion:  config.Version,
		ManufacturerName: config.Manufacturer,
		SerialNumber:     config.Serial,
		ModelNumber:      config.Model,
	}

	tlv := htest.NewTLVBuffer(
		2 + len(report.FirmwareVersion) +
			2 + len(report.ManufacturerName) +
			2 + len(report.SerialNumber) +
			2 + len(report.ModelNumber))

	tlv.Begin()
	tlv.EncodeTLVString(htest.FirmwareID, report.FirmwareVersion)
	tlv.EncodeTLVString(htest.ManufacturerID, report.ManufacturerName)
	tlv.EncodeTLVString(htest.SerialID, report.SerialNumber)
	tlv.EncodeTLVString(htest.ModelID, report.ModelNumber)

	req := coap.Message{
		Type:      coap.Confirmable,
		Code:      coap.POST,
		MessageID: 1234,
		Token:     []byte{1, 2, 3, 4, 5, 6, 7, 8},
		Payload:   tlv.Buffer(),
	}
	req.SetPathString("/u")

	res, err := conn.Send(req)
	if err != nil {
		return err
	}

	response, err := htest.DecodeSimpleFOTAResponse(res.Payload)
	if err != nil {
		return err
	}

	fmt.Printf("Host: %s, Port: %d, Path: %s, Image: %t\n", response.Host, response.Port, response.Path, response.ImageAvailable)

	if !response.ImageAvailable && config.NoNew {
		fmt.Println("No new expected and no firmware available")
		return nil
	}
	if config.NoNew {
		return errors.New("did not expect any firmware to be available")
	}
	if !response.ImageAvailable {
		return errors.New("no image is available")
	}

	fmt.Printf("Will download image from coap://%s:%d/%s\n", response.Host, response.Port, response.Path)
	fwConn, err := coap.Dial("udp", fmt.Sprintf("%s:%d", response.Host, response.Port))
	if err != nil {
		return err
	}
	// Download image
	fwReq := coap.Message{
		Type:      coap.Confirmable,
		Code:      coap.GET,
		MessageID: 4321,
	}
	fwReq.SetPathString(fmt.Sprintf("/%s", response.Path))

	fwRes, err := fwConn.Send(fwReq)
	if err != nil {
		return err
	}
	if fwRes.Code != coap.Content {
		return fmt.Errorf("unexpected response: %s", fwRes.Code.String())
	}
	// Note that this won't work properly since the CoAP library has no support for blockwise
	// transfers but we won't be using the result anyways. It might time out on
	// the server but that's OK.
	fmt.Printf("Downloaded image. Size = %d bytes\n", len(fwRes.Payload))

	return nil
}
