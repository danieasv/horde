package utils

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
	"testing"
)

func TestNtoaAtonConversion(t *testing.T) {
	ip1 := net.ParseIP("1.2.3.4")
	numip1 := 0x01020304
	ip2 := net.ParseIP("1.2.3.4")
	ip3 := net.ParseIP("1.2.3.5")

	if AtonIPv4(ip1) != uint32(0x01020304) {
		t.Fatalf("Expected %v to be equal to %08x but it was %08x", ip1, numip1, AtonIPv4(ip1))
	}
	if AtonIPv4(ip1) != AtonIPv4(ip2) {
		t.Fatalf("Expected %v (%v) to be equal to %v (%v)", AtonIPv4(ip1), ip1, AtonIPv4(ip2), ip2)
	}

	if AtonIPv4(ip1) == AtonIPv4(ip3) {
		t.Fatal(ip1, " == ", ip3)
	}
}
