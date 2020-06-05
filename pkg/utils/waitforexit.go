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
	"os"
	"os/signal"
	"syscall"

	"github.com/ExploratoryEngineering/logging"
)

// Make this available for tests
var sigch chan os.Signal

func init() {
	sigch = make(chan os.Signal, 2)
}

// WaitForSignal waits for a signal to terminate
func WaitForSignal() {
	logging.Debug("Waiting for kill signal")
	terminator := make(chan bool)

	signal.Notify(sigch, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-sigch
		logging.Debug("Caught signal '%v'", sig)
		terminator <- true
	}()
	<-terminator
}

// SendInterrupt sends an interrupt signal to the waiting channel
func SendInterrupt() {
	select {
	case sigch <- os.Interrupt:
		// ok
	default:
		// ignore
	}
}

// GetSignalChannel returns the signal channel. This is for testing.
func GetSignalChannel() chan os.Signal {
	return sigch
}
