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
	"errors"
	"time"

	"github.com/eesrc/horde/pkg/model"
)

var (
	// ErrUnknownOutputType is returned when the output type is
	// unknown or incorrect-
	ErrUnknownOutputType = errors.New("unknown output type")
	// ErrInvalidConfig is returned when one or more of the output's
	// configuration parameters are invalid.
	ErrInvalidConfig = errors.New("invalid output config")
)

// Output defines the interface for the various outputs. Outputs forwards data
// from Horde to external servers. Outputs can be stateless (webhooks) or have
// a connection state but they behave similarly. Messages will be cached for
// minor network glitches.
type Output interface {
	// Validate verifies the configuraton for the output. Error messages that
	// should be provided to the end user is returned as the 2nd parameter.
	// On success the error return value is nil.
	Validate(config model.OutputConfig) (model.ErrorMessage, error)
	// Start launches the output. This is non blocking, ie if the output
	// must connect to a remote server or perform some sort initialization it
	// will do so in a separate goroutine. The output will stop automatically
	// when the message channel is closed. If the message channel is closed
	// the output should attempt empty any remaining messages in the queue.
	Start(config model.OutputConfig, collectionFieldMask model.FieldMask, systemFieldMask model.FieldMask, message <-chan interface{})
	// Stop halts the output. Any buffered messages that can't be sent during
	// the timeout will be discarded by the output. When the Stop call returns
	// the output has stopped.
	Stop(timeout time.Duration)
	// Logs returns end user logs for the output.
	Logs() []model.OutputLogEntry
	// Status reports the internal status of the forwarder.
	Status() model.OutputStatus
}

// NewOutput creates a new output. It will be running until it shuts down.
func NewOutput(outputType string) (Output, error) {
	return makeOutput(outputType)
}
