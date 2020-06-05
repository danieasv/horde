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
import "fmt"

// FirmwareUpdateState is the state of the firmware update
type FirmwareUpdateState byte

const (
	// Idle ie the client is idle before downloading or update
	Idle = FirmwareUpdateState(0)
	// Downloading ie the client is downloading the firmware
	Downloading = FirmwareUpdateState(1)
	// Downloaded ie firmware is downloaded but not applied
	Downloaded = FirmwareUpdateState(2)
	// Updating ie firmware is currently updated
	Updating = FirmwareUpdateState(3)
)

func (s FirmwareUpdateState) String() string {
	switch s {
	case Idle:
		return "Idle"
	case Downloading:
		return "Downloading"
	case Downloaded:
		return "Downloaded"
	case Updating:
		return "Updating"
	}
	return fmt.Sprintf("Unknown(%d)", s)
}

// FirmwareUpdateResult is the result of the firmware update
type FirmwareUpdateResult byte

const (
	// Success updating firmware
	Success = FirmwareUpdateResult(1)
	// NotEnoughFlash not enough flash to update
	NotEnoughFlash = FirmwareUpdateResult(2)
	// OutOfMemory while updating
	OutOfMemory = FirmwareUpdateResult(3)
	// ConnectionLost while receiving or reading firmware
	ConnectionLost = FirmwareUpdateResult(4)
	// IntegrityCheckFailure on downloaded firmware
	IntegrityCheckFailure = FirmwareUpdateResult(5)
	// UnsupportedPackageType when checking downloaded firmware
	UnsupportedPackageType = FirmwareUpdateResult(6)
	// InvalidURI when attempting download
	InvalidURI = FirmwareUpdateResult(7)
	// UpdateFailed when updating firmware (probably one of the most used errors TBH)
	UpdateFailed = FirmwareUpdateResult(8)
	// UnsupportedProtocol in firmware URI
	UnsupportedProtocol = FirmwareUpdateResult(9)
)

func (r FirmwareUpdateResult) String() string {
	switch r {
	case Success:
		return "Success"
	case NotEnoughFlash:
		return "NotEnoughFlash"
	case OutOfMemory:
		return "OutOfMemory"
	case ConnectionLost:
		return "ConnectionLost"
	case IntegrityCheckFailure:
		return "IntegrityCheckFailure"
	case UnsupportedPackageType:
		return "UnsupportedPackageType"
	case InvalidURI:
		return "InvalidURI"
	case UpdateFailed:
		return "UpdatedFailed"
	case UnsupportedProtocol:
		return "UnsupportedProtocol"
	default:
		return fmt.Sprintf("Unknown(%d)", r)
	}
}

// UpdateProtocol is the protocols the device supports for URIs
type UpdateProtocol byte

const (
	// CoAP over UDP, ie regular coap:// URIs (RFC 7252)
	CoAP = UpdateProtocol(0)
	// CoAPS over DTLS, ie coaps:// URIs (RFC 7252)
	CoAPS = UpdateProtocol(1)
	// HTTP protocol support, ie http:// URIs (RFC 7230)
	HTTP = UpdateProtocol(2)
	// HTTPS protocol support, ie https:// URIs (RFC 7230)
	HTTPS = UpdateProtocol(3)
	// CoAPTCP is CoAP over TCP , ie coap+tcp:// URIs (RFC 7230)
	CoAPTCP = UpdateProtocol(4)
	// CoAPTLS is CoAP over TLS, ie coaps+tcp:// URIs (RFC 7230)
	CoAPTLS = UpdateProtocol(5)
)

// DeliveryMethod is the supported firmware delivery methods
type DeliveryMethod byte

const (
	// PullDelivery is delivery via package URI field
	PullDelivery = DeliveryMethod(0)
	// PushDelivery is delivery via the package resource
	PushDelivery = DeliveryMethod(1)
	// PushAndPullDelivery are both delivery methods
	PushAndPullDelivery = DeliveryMethod(2)
)

// FirmwareUpdate is the response message from the device when object 5 is
// queried. The object isn't complete.
// The object is defined in http://www.openmobilealliance.org/release/LightweightM2M/V1_1-20180710-A/OMA-TS-LightweightM2M_Core-V1_1-20180710-A.pdf
// section E.6
type FirmwareUpdate struct {
	// PackageURI is the URI for the package that the device will download/have
	// downloaded. Object ID is 1.
	PackageURI string
	// State is the current state of the firmware update wrt the URI or uploaded
	// firmware. Object ID is 3.
	State FirmwareUpdateState
	// UpdateResult is the result after the firmware update. Object ID is 5
	UpdateResult FirmwareUpdateResult
	// PackageName is the name of the firmware package. Object ID is 6
	PackageName string
	// PackageVersion is the version of the package. Object ID is 7
	PackageVersion string
	// SupportedProtocols is the list of protocols the firmware supports. Object ID is 8
	SupportedProtocols []UpdateProtocol
	// SupportedDelivery is the delivery method supported by the device. Object ID is 9
	SupportedDelivery DeliveryMethod
}

// SetSupportedProtocols reads the object at /5/0/8
func (f *FirmwareUpdate) SetSupportedProtocols(b TLVBuffer) {
	protocols := b.GetPayloadList(FirmwareSupportedProtocolsID)
	if protocols != nil {
		f.SupportedProtocols = make([]UpdateProtocol, len(protocols))
		for i, v := range protocols {
			f.SupportedProtocols[i] = UpdateProtocol(v.Byte())
		}
	}
}

// SetSupportedDelivery reads the object at /5/0/9
func (f *FirmwareUpdate) SetSupportedDelivery(b TLVBuffer) {
	delivery := b.GetPayload(FirmwareDeliveryMethodID)
	if delivery != nil {
		f.SupportedDelivery = DeliveryMethod(delivery.Byte())
	}
}

// SetState sets the state field
func (f *FirmwareUpdate) SetState(b TLVBuffer) {
	state := b.GetPayload(FirmwareStateID)
	if state != nil {
		f.State = FirmwareUpdateState(state.Byte())
	}
}

// SetUpdateResult sets the UpdateResult field
func (f *FirmwareUpdate) SetUpdateResult(b TLVBuffer) {
	result := b.GetPayload(FirmwareUpdateResultID)
	if result != nil {
		f.UpdateResult = FirmwareUpdateResult(result.Byte())
	}
}

// SetPackageName sets the package name from the buffer
func (f *FirmwareUpdate) SetPackageName(b TLVBuffer) {
	name := b.GetPayload(FirmwarePackageNameID)
	if name != nil {
		f.PackageName = name.String()
	}
}

// SetPackageVersion sets the package version
func (f *FirmwareUpdate) SetPackageVersion(b TLVBuffer) {
	ver := b.GetPayload(FirmwarePackageVersionID)
	if ver != nil {
		f.PackageVersion = ver.String()
	}
}

// NewFirmwareUpdate creates and initializes a FirmwareUpdate type with values from
// an ObjectInstanceTLV buffer
func NewFirmwareUpdate(b TLVBuffer) FirmwareUpdate {

	uri := b.GetPayload(FirmwareURIID)
	state := b.GetPayload(FirmwareStateID)
	result := b.GetPayload(FirmwareUpdateResultID)
	name := b.GetPayload(FirmwarePackageNameID)
	version := b.GetPayload(FirmwarePackageVersionID)
	delivery := b.GetPayload(FirmwareDeliveryMethodID)

	ret := FirmwareUpdate{}
	if uri != nil {
		ret.PackageURI = uri.String()
	}
	if state != nil {
		ret.State = FirmwareUpdateState(state.Byte())
	}
	if result != nil {
		ret.UpdateResult = FirmwareUpdateResult(result.Byte())
	}
	if name != nil {
		ret.PackageName = name.String()
	}
	if version != nil {
		ret.PackageVersion = version.String()
	}
	if delivery != nil {
		ret.SupportedDelivery = DeliveryMethod(delivery.Byte())
	}
	protocols := b.GetPayloadList(0, FirmwareSupportedProtocolsID)
	if protocols != nil {
		ret.SupportedProtocols = make([]UpdateProtocol, len(protocols))
		for i, v := range protocols {
			ret.SupportedProtocols[i] = UpdateProtocol(v.Byte())
		}
	}
	return ret
}
