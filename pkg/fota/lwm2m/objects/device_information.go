package objects

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

// DeviceInformation is the response message from the device when object 3 is
// queried. The object isn't complete.
// The object is defined in http://www.openmobilealliance.org/release/LightweightM2M/V1_1-20180710-A/OMA-TS-LightweightM2M_Core-V1_1-20180710-A.pdf
// section E.4
type DeviceInformation struct {
	// Manufacturer is the human readable name. Resource 0
	Manufacturer string
	// ModelNumber is the module identifier. Resource 1
	ModelNumber string
	// SerialNumber is the device's serial number. Resource 2
	SerialNumber string
	// FirmwareVersion  is the current firmware version of the device. Resource 3
	FirmwareVersion string
	// BatteryLevel is the current battery level 0-100. Resource 9
	BatteryLevel byte
}

// NewDeviceInformation creates and initializes a new DeviceInformation type
// with values read from a byte buffer
func NewDeviceInformation(b TLVBuffer) DeviceInformation {
	mf := b.GetPayload(ManufacturerID)
	mn := b.GetPayload(ModelNumberID)
	sn := b.GetPayload(SerialNumberID)
	fw := b.GetPayload(FirmwareVersionID)
	bl := b.GetPayload(BatteryLevelID)

	ret := DeviceInformation{}
	if mf != nil {
		ret.Manufacturer = mf.String()
	}
	if mn != nil {
		ret.ModelNumber = mn.String()
	}
	if sn != nil {
		ret.SerialNumber = sn.String()
	}
	if fw != nil {
		ret.FirmwareVersion = fw.String()
	}
	if bl != nil {
		ret.BatteryLevel = bl.Byte()
	}
	return ret
}

// Buffer encodes the device information block to a TLV buffer
func (d *DeviceInformation) Buffer() []byte {
	var resources []byte

	resources = append(resources, EncodeString(ManufacturerID, d.Manufacturer)...)
	resources = append(resources, EncodeString(ModelNumberID, d.ModelNumber)...)
	resources = append(resources, EncodeString(SerialNumberID, d.SerialNumber)...)
	resources = append(resources, EncodeString(FirmwareVersionID, d.FirmwareVersion)...)
	resources = append(resources, EncodeBytes(BatteryLevelID, []byte{d.BatteryLevel})...)

	return resources
}
