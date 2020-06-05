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
	"fmt"
	"testing"
)

func TestEndpointCheker(t *testing.T) {
	e := newEndpointChecker("@@https::\\\\foo?foo?foo")
	if e.IsValidHost() || e.IsValidHTTPURL() || e.IsValidMQTTEndpoint() || e.IsSSLScheme() {
		t.Fatal("Invalid URL")
	}

	e = newEndpointChecker("tcp://example.com:1883")
	if !e.IsValidMQTTEndpoint() {
		t.Fatal("Should be a valid MQTT endpoint")
	}

	if e.IsValidHTTPURL() {
		t.Fatal("tcp:// isn't a valid HTTP URL")
	}

	if e.IsSSLScheme() {
		t.Fatal("This isn't a SSL scheme")
	}

	e = newEndpointChecker("ssl://example.com:8883")
	if !e.IsValidMQTTEndpoint() {
		t.Fatal("Should be a valid MQTT endpoint")
	}

	e = newEndpointChecker("ssl://example.com")
	if e.IsValidMQTTEndpoint() {
		t.Fatal("Shouldn't be a valid MQTT endpoint")
	}

	e = newEndpointChecker("ssl://")
	if e.IsValidMQTTEndpoint() {
		t.Fatal("Shouldn't be valid")
	}
	e = newEndpointChecker("http://")
	if e.IsValidHTTPURL() {
		t.Fatal("Not a valid HTTO endpoint")
	}
	e = newEndpointChecker("http://localhost")
	if !e.IsValidHTTPURL() {
		t.Fatal("Should be a valid HTTP endpoint")
	}
	if e.IsValidMQTTEndpoint() {
		t.Fatal("Should not be a valid MQTT endpoint")
	}
	if e.Host() != "localhost" {
		t.Fatal("Incorrect host")
	}

	e = newEndpointChecker("tcp://example.com:1883/")
	if e.IsValidMQTTEndpoint() {
		t.Fatal("Path should not be included in MQTT URLs")
	}
	e = newEndpointChecker("tcp://example.com:1883/foof")
	if e.IsValidMQTTEndpoint() {
		t.Fatal("Path should not be included in MQTT URLs (II)")
	}
}

func TestEndpointHost(t *testing.T) {
	verifyInvalid := func(t *testing.T, host string, ipv6 bool) {
		ep := fmt.Sprintf("http://%s", host)
		e := newEndpointChecker(ep)
		if !e.IsValidHTTPURL() {
			t.Fatalf("Not a valid MQTT endpoint: %s", ep)
		}
		// ipv6 addresses are ... complicated
		if !ipv6 {
			if e.Host() != host {
				t.Fatalf("Host should be unchanged but '%s' != '%s'", host, e.Host())
			}
		}
		if e.IsValidHost() {
			t.Fatalf("Host %s should not be valid", host)
		}
	}

	verifyInvalid(t, "localhost", false)
	verifyInvalid(t, "127.0.0.1", false)
	verifyInvalid(t, "example.com", false)
	verifyInvalid(t, "10.0.1.10", false)
	verifyInvalid(t, "ip-172-16-0-100.eu-west-1.compute.internal.", false)
	verifyInvalid(t, "172.16.0.100", false)
	verifyInvalid(t, "172.24.0.100", false)
	verifyInvalid(t, "172.25.0.5", false)
	verifyInvalid(t, "192.168.0.1", false)
	verifyInvalid(t, "noapi.nbiot.telenor.io", false)
	verifyInvalid(t, "1:1:1", true)
	verifyInvalid(t, "2001:2::", true)
	verifyInvalid(t, "something-or-other.internal", true)
	verifyInvalid(t, "something-or-other.internal.", true)
	verifyInvalid(t, "whatever.example.org", true)
	verifyInvalid(t, "143.204.47.78.1", false)

	// Valid hosts: ipv6.google.com (resolves to ipv6 only hosts)
	verifyValid := func(t *testing.T, host string) {
		ep := fmt.Sprintf("http://%s", host)
		e := newEndpointChecker(ep)
		if !e.IsValidHTTPURL() {
			t.Fatal("Not a valid MQTT endpoint: ", ep)
		}
		if !e.IsValidHost() {
			t.Fatal("Expected host to be valid: ", host)
		}
	}
	verifyValid(t, "143.204.47.78")
	verifyValid(t, "telenordigital.com")
	verifyValid(t, "amazon.com")
	verifyValid(t, "ipv6.google.com")
	//	verifyValid(t, "2a00:1450:400f:809::200e")
	verifyValid(t, "172.100.0.100")

}
