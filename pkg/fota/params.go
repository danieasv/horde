package fota

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
	"net/url"
	"strconv"
	"time"
)

// Parameters is the config parameters for the FOTA endpoints. This includes
// both LwM2M and simple FOTA.
type Parameters struct {
	// FirmwareEndpoint is the endpoint for the firmware resource. This is used
	// both by the LwM2M FOTA and the simple FOTA processes.
	FirmwareEndpoint string `param:"desc=CoAP firmware endpoint;default=coap://172.16.15.14:5683/fw"`

	//LWM2MTimeout is the timeout for LwM2M requests. It is set quite high since
	// the devices can be slow to respond. The network latency can be as high
	// as 5 seconds.
	LWM2MTimeout time.Duration `param:"desc=LwM2M request timeout;default=30s"`

	// DownloadTmeout is the timeout for a firmware download.
	// 30 minutes is a rough guess (it was 5 but it's way to short, downloading
	// an image for nRf91 took 14 minutes. 30 minutes might be more like it)
	// This is radically different from regular
	// CoAP responses towards the devices since they are writing to flash
	// and will generally be very busy at this point.
	// N2 offloads is *really* slow and might be upwards of 30-40 minutes.
	// Bumping to 1h.
	DownloadTimeout time.Duration `param:"desc=Firmware download timeout;default=60m"`

	// LWM2MPollInterval is the polling interval for the firmware state during
	// the download. The default is 30s which is about the regular observe
	// intervals found in Zephyr. Decrease to speed up checks (but more polling
	// uses more power), lower to make the checks slower. A download over a
	// slow NB-IoT link might be 4800bps or less, depending on the configuration
	// of the module. uBlox N2 uses a 9600 baud UART to send and receive data
	// and the data is encoded as hex digits which makes the link act like a
	// 2400 baud modem. Fortunately this link is quicker for nRF91 (more like
	// a few hundred kbps but a firmware download can still be measured in minutes
	// not seconds.)
	LWM2MPollInterval time.Duration `param:"desc=Polling interval for firmware state during download;default=30s"`
}

// GetFirmwareHostPortPath splits the firmware endpoint into its separate components
func (p *Parameters) GetFirmwareHostPortPath() (string, int, string, error) {
	u, err := url.Parse(p.FirmwareEndpoint)
	if err != nil {
		return "", 0, "", err
	}
	portStr := u.Port()
	if portStr == "" {
		portStr = "5683"
	}
	port, err := strconv.ParseInt(portStr, 10, 32)
	if err != nil {
		return "", 0, "", err
	}
	path := u.EscapedPath()
	if len(path) > 1 {
		path = path[1:]
	}
	return "" + u.Hostname(), int(port), path, nil
}
