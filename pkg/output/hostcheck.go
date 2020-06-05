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
	"net"
	"net/url"
	"strings"

	"github.com/ExploratoryEngineering/logging"
)

// endpointChecker checks endpoints and URLs for private IPs and invalid host
// names. Invalid domains include:
//
//    * .test
//    * .example
//    * .invalid
//    * .localhost
//    * example.com
//    * example.net
//    * example.org
//
// Invalid subnets include:
//
// IPv4:
//     10.0.0.0        -   10.255.255.255  (10/8 prefix)
//     172.16.0.0      -   172.31.255.255  (172.16/12 prefix)
//     192.168.0.0     -   192.168.255.255 (192.168/16 prefix)
//     127.0.0.0/8
//
//  IPv6 (this list might grow):https://www.iana.org/assignments/iana-ipv6-special-registry/iana-ipv6-special-registry.xhtml
//      ::1/128
//      ::ffff:0:0/96
//      100::/64
//      2001::/23
//      2001:2::/48
//      fc00::/7
//      fe80::/10
//
// Ports and IPv6 addresses in MQTT is broken.
//
type endpointChecker struct {
	endpoint string
	u        *url.URL
}

func newEndpointChecker(endpoint string) endpointChecker {
	// ignore error since u wil
	u, err := url.Parse(endpoint)
	if err != nil {
		logging.Warning("Invalid URL (%s): %v", endpoint, err)
		u = nil
	}
	return endpointChecker{endpoint: endpoint, u: u}
}

// IsValidMQTTEndpoint checks if the endpoint *format* is valid. The
// actual host must be checked separately.
func (e *endpointChecker) IsValidMQTTEndpoint() bool {
	if e.u == nil {
		return false
	}
	if e.u.Host == "" {
		return false
	}
	if e.u.Scheme != "tcp" && e.u.Scheme != "ssl" {
		return false
	}
	// Note: the port breaks down for ipv6 addresses
	if e.u.Port() == "" {
		return false
	}
	if e.u.Path != "" {
		return false
	}
	return true
}

// IsSSLScheme checks if the endpoint uses the "ssl" scheme, ie TLS for MQTT
func (e *endpointChecker) IsSSLScheme() bool {
	if e.u == nil {
		return false
	}
	return e.u.Scheme == "ssl"
}

// Host returns the host
func (e *endpointChecker) Host() string {
	if e.u == nil {
		return ""
	}
	h, _, err := net.SplitHostPort(e.u.Host)
	if err != nil {
		return e.u.Host
	}
	return h
}

var cidrList = []string{
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"127.0.0.0/8",
	"::1/128",
	"100::/64",
	"2001::/23",
	"2001:2::/48",
	"fc00::/7",
	"fe80::/10",
}
var invalidDomains = []string{
	"example.com",
	"example.org",
	"example.net",
	".test",
	".example",
	".invalid",
	".localhost",
}

var invalidCIDRs []*net.IPNet

func init() {
	for _, s := range cidrList {
		_, net, err := net.ParseCIDR(s)
		if err != nil {
			logging.Warning("Unable to parse CDIR %s: %v", s, err)
			continue
		}
		invalidCIDRs = append(invalidCIDRs, net)
	}
}

// DisableLocalhostChecks turns OFF checks for outputs to localhost
// This shoudln't be enabled in production but is very useful when testing
// on your local computer or during unit testing where servers are running
// locally
func DisableLocalhostChecks() {
	// TODO(stalehd) consider removing ::1/128 since it resolves
	// to "localhost" as well.
	logging.Error("Host checks for local IPv4 endpoints are disabled!")
	for _, v := range invalidCIDRs {
		logging.Error("CIDR check for %s is DISABLED", v.String())
	}
	invalidCIDRs = make([]*net.IPNet, 0)
}

// IsValidHost checks if the host part of the endpoint/URL is a valid host on
// the internet.
func (e *endpointChecker) IsValidHost() bool {
	ephost := e.Host()
	for _, domain := range invalidDomains {
		if strings.HasSuffix(ephost, domain) {
			return false
		}
	}
	host, err := net.LookupHost(ephost)
	if err != nil {
		return false
	}
	for _, addr := range host {
		ip := net.ParseIP(addr)
		if ip == nil {
			logging.Info("Invalid IP: %s", addr)
			return false
		}
		for _, v := range invalidCIDRs {
			if v.Contains(ip) {
				logging.Info("Invalid CIDR: %s (ip=%s)", v.String(), ip.String())
				return false
			}
		}
	}
	return true
}

// IsValidHTTPURL returns true if the endpoint is a properly *formatted* HTTP URL.
// IsValidHost() might return false.
func (e *endpointChecker) IsValidHTTPURL() bool {
	if e.u == nil {
		return false
	}
	if e.u.Host == "" {
		return false
	}
	if e.u.Scheme != "http" && e.u.Scheme != "https" {
		return false
	}
	return true
}
