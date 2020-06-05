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
	"flag"
	"math/rand"
	"net"
	"sync/atomic"
	"time"

	"github.com/ExploratoryEngineering/logging"
)

var (
	destinationAddr = flag.String("destination_addr", "127.0.0.1:31415", "Destination address for UDP packets")
	sleepTime       = flag.Duration("time_between", 1000*time.Millisecond, "Time between messages")
)

func main() {
	flag.Parse()
	logging.EnableStderr(true)
	logging.SetLogLevel(logging.DebugLevel)

	conn, err := net.Dial("udp", *destinationAddr)
	if err != nil {
		logging.Error("Unable to create UDP socket to %s -- %v", *destinationAddr, err)
		return
	}
	defer conn.Close()

	payload := make([]byte, 64)
	rand.Read(payload)
	created := int64(0)

	// Print total on ctrl+c
	logging.Info("Sending data")
	go func() {
		last := int64(0)
		for {
			current := atomic.LoadInt64(&created)
			count := current - last
			last = current
			logging.Debug("%d msg/sec sent (%d total)", count, current)
			time.Sleep(time.Second)
		}
	}()
	for {
		_, err := conn.Write(payload)
		if err != nil {
			logging.Error("Error sending UDP data to %s - %v", *destinationAddr, err)
			time.Sleep(time.Second)
		}
		if err == nil {
			atomic.AddInt64(&created, 1)
		}
		time.Sleep(*sleepTime)
	}
}
