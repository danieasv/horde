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
	"net"

	"github.com/eesrc/horde/pkg/apn/radius/threegpp"
	"layeh.com/radius"

	"layeh.com/radius/rfc2865"
)

const (
	// HandlerNotRegisteredErrorMsg is used when there is no
	// AccessRequestHandler has been set
	HandlerNotRegisteredErrorMsg = "Server error 1"
	// IPAddressInvalid is used when the AccessRequestHandler returns
	// an invalid IP address such as 0.0.0.0
	IPAddressInvalid = "Server error 2"
)

// AccessRequestHandlerFunc allows a function to implement AccessRequestHandler
type AccessRequestHandlerFunc func(AccessRequest) AccessResponse

// AccessRequest contains all the attributes we might be interested in
// from the 4G network.
type AccessRequest struct {
	Username         string
	Password         string
	NASIPAddress     net.IP
	NASIdentifier    string
	IMSI             string
	IMSIMccMnc       string
	UserLocationInfo []byte
	MSTimezone       []byte
	IMEISV           string
}

// AccessResponse is used by the handler to signal what the Radius
// server should do with the request.  If Accept is true the
// Access-Request is granted and a Access-Accept is returned. When
// accepting IPAddress and Netmask must be set.
//
// If Accept is false an Access-Reject message is sent. The
// RejectMessage should be set.
type AccessResponse struct {
	Accept        bool
	IPAddress     net.IP
	RejectMessage string
}

func accessRequestFromPacket(p *radius.Packet) AccessRequest {
	return AccessRequest{
		Username:      rfc2865.UserName_GetString(p),
		Password:      rfc2865.UserPassword_GetString(p),
		NASIPAddress:  rfc2865.NASIPAddress_Get(p),
		NASIdentifier: rfc2865.NASIdentifier_GetString(p),
		IMSI:          string(threegpp.ThreeGPPIMSI_Get(p)),
		IMSIMccMnc:    threegpp.ThreeGPPIMSIMCCMNC_GetString(p),
		MSTimezone:    threegpp.ThreeGPPMSTimeZone_Get(p),
		IMEISV:        threegpp.ThreeGPPIMEISV_GetString(p),
	}
}
