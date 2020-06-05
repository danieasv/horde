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
	"strconv"
	"testing"

	"github.com/eesrc/horde/pkg/model"
)

func TestUDPOutput(t *testing.T) {
	DisableLocalhostChecks()
	// Create listener first to get a free UDP port
	pc, err := net.ListenPacket("udp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	host, port, err := net.SplitHostPort(pc.LocalAddr().String())
	if err != nil {
		t.Fatal(err)
	}
	p, _ := strconv.ParseInt(port, 10, 32)
	config := model.OutputConfig{
		"host": host,
		"port": float64(p), // because the JSON unmarshaller is weird
	}
	quit := make(chan bool)
	defer pc.Close()
	go func() {
		buf := make([]byte, 1024)
		for {
			_, _, err := pc.ReadFrom(buf)
			if err != nil {
				continue
			}
			select {
			case <-quit:
				return
			default:
			}
		}
	}()
	outputTests(newUDP(), config, t)
	quit <- true
}
