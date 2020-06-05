package lwm2m

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

// These are the paths to the various resources in the device's LwM2M CoAP
// server
const (
	DeviceInformationPath        = "/3/0"
	FirmwareImageURIPath         = "/5/0/1"
	FirmwareImagePath            = "/5/0/0"
	FirmwareUpdatePath           = "/5/0/2"
	FirmwareStatePath            = "/5/0/3"
	FirmwareUpdateResultPath     = "/5/0/5"
	FirmwarePackageNamePath      = "/5/0/6"
	FirmwarePackageVersionPath   = "/5/0/7"
	SupportedProtocolsPath       = "/5/0/8"
	SupportedDeliveryMethodsPath = "/5/0/9"
	RebootPath                   = "/3/4"
)
