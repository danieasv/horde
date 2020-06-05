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
	"fmt"
	"net"
	"strings"
	"time"
)

const (
	port = 4711
)

func main() {
	hostPort := fmt.Sprintf("127.0.0.1:%d", port)

	addr, err := net.ResolveUDPAddr("udp", hostPort)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Listening on %s\n", hostPort)
	serverConn, err := net.ListenUDP("udp", addr)
	if err != nil {
		panic(err)
	}
	defer serverConn.Close()
	for {
		serverConn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		buf := make([]byte, 4096)
		n, addr, err := serverConn.ReadFromUDP(buf)
		if err != nil {
			continue
		}
		if n > 0 {
			fmt.Printf("Got %d bytes from %s: %v (%s)\n", n, addr, buf[0:n], strings.TrimSpace(string(buf[0:n])))
		}
	}
}
