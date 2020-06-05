package main

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
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/eesrc/horde/pkg/apn/radius/threegpp"
	nbiot "github.com/telenordigital/nbiot-go"
	"layeh.com/radius"

	// Use
	"layeh.com/radius/rfc2865"
)

func msisdnForDevice(device nbiot.Device) string {
	return fmt.Sprintf("47%08s", device.IMSI)
}

func imeisvForDevice(device nbiot.Device) string {
	return fmt.Sprintf("%s00", device.IMEI)
}

// Constants for the "I don't care" values
const (
	radiusPassword                   = "password"
	nasIPAddress                     = "192.168.0.1"
	nasIdentifier                    = "NAS"
	callingStationID                 = "cid"
	serviceType                      = 2
	framedProtocol                   = 7
	nasPortType                      = 18
	nasPort                          = 1077866
	threeGPPGGSNAddress              = "217.148.144.78"
	threeGPPSGSNAddress              = "217.148.144.73"
	threeGPPNSAPI                    = "6"
	threeGPPChargingID               = 488403561
	threeGPPChargingCharacteristics  = "0100"
	threeGPPRATType                  = 6
	threeGPPSelectionMode            = "1"
	threeGPPMSTimeZone               = "@"
	threeGPPGPRSNegotiatedQoSprofile = "08-58090007A1200"
	threeGPPPDPType                  = 0
	threeGPPNegotiatedDSCP           = 0
)

var mutex sync.Mutex

// do the request to the RADIUS server. Use the regular network interface
// for this. Assign the IP to the device.
func doRADIUSRequest(device nbiot.Device, config args) bool {
	mutex.Lock()
	defer mutex.Unlock()
	packet := radius.New(radius.CodeAccessRequest, []byte(config.RADIUSSharedSecret))

	rfc2865.UserName_SetString(packet, msisdnForDevice(device))
	rfc2865.UserPassword_SetString(packet, radiusPassword)
	rfc2865.NASIPAddress_Add(packet, net.ParseIP(nasIPAddress))
	rfc2865.NASIdentifier_Add(packet, []byte(nasIdentifier))
	rfc2865.CallingStationID_Add(packet, []byte(callingStationID))
	rfc2865.ServiceType_Add(packet, rfc2865.ServiceType(serviceType))
	rfc2865.FramedProtocol_Add(packet, rfc2865.FramedProtocol(framedProtocol))
	rfc2865.NASPortType_Add(packet, rfc2865.NASPortType(nasPortType))
	rfc2865.NASPort_Add(packet, rfc2865.NASPort(nasPort))

	threegpp.ThreeGPPIMSI_AddString(packet, device.IMSI)
	threegpp.ThreeGPPGGSNAddress_Add(packet, net.ParseIP(threeGPPGGSNAddress))
	threegpp.ThreeGPPNSAPI_Add(packet, []byte(threeGPPNSAPI))
	threegpp.ThreeGPPChargingID_Add(packet, threegpp.ThreeGPPChargingID(threeGPPChargingID))
	threegpp.ThreeGPPChargingCharacteristics_Add(packet, []byte(threeGPPChargingCharacteristics))
	threegpp.ThreeGPPSGSNAddress_Add(packet, net.ParseIP(threeGPPSGSNAddress))
	threegpp.ThreeGPPGGSNAddress_Add(packet, net.ParseIP(threeGPPGGSNAddress))
	threegpp.ThreeGPPRATType_Add(packet, threegpp.ThreeGPPRATType(threeGPPRATType))
	threegpp.ThreeGPPSelectionMode_AddString(packet, threeGPPSelectionMode)
	threegpp.ThreeGPPIMEISV_AddString(packet, imeisvForDevice(device))
	threegpp.ThreeGPPMSTimeZone_Add(packet, []byte(threeGPPMSTimeZone))
	threegpp.ThreeGPPGPRSNegotiatedQoSProfile_Add(packet, []byte(threeGPPGPRSNegotiatedQoSprofile))
	threegpp.ThreeGPPPDPType_Add(packet, threegpp.ThreeGPPPDPType(threeGPPPDPType))
	threegpp.ThreeGPPNegotiatedDSCP_Add(packet, threegpp.ThreeGPPNegotiatedDSCP(threeGPPNegotiatedDSCP))
	threegpp.ThreeGPPUserLocationInfo_AddString(packet, "8242f2109dd142f2100102da67")

	log.Printf("Sending radius request to %s for device %s", config.RadiusEP, device.ID)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	response, err := radius.Exchange(ctx, packet, config.RadiusEP)
	if err != nil {
		log.Printf("Error performing RADIUS request for device %s: %v\n", device.ID, err)
		return false
	}
	if response.Code != radius.CodeAccessAccept {
		log.Printf("Did not get AccessAccept response from server for device %s", device.ID)
		return false
	}
	ipaddr := rfc2865.FramedIPAddress_Get(response)
	log.Printf("IP address for device %s is %s\n", device.ID, ipaddr.String())
	device.Tags["ip"] = ipaddr.String()
	return true
}
