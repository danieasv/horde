package model

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
	"database/sql/driver"
	"encoding/json"
	"errors"
	"reflect"
	"time"
)

// OutputKey is the identifier for outputs
type OutputKey storageKey

// NewOutputKeyFromString creates a new OutputKey  from a string
// representation
func NewOutputKeyFromString(id string) (OutputKey, error) {
	k, err := newKeyFromString(id)
	return OutputKey(k), err
}

// String returns the string representation of the DeviceKey instance
func (d OutputKey) String() string {
	return storageKey(d).String()
}

// OutputConfig is a map of configuration values
type OutputConfig map[string]interface{}

// HasParameterOfType checks if the parameter exists and if type type is correct
func (o OutputConfig) HasParameterOfType(name string, expectedType reflect.Kind) (bool, bool) {
	v, exists := o[name]
	if !exists || v == nil {
		return false, false
	}
	correctType := reflect.TypeOf(v).Kind() == expectedType
	return exists, correctType

}

// NewOutputConfig creates a new output configuration
func NewOutputConfig() OutputConfig {
	return make(OutputConfig)
}

// Scan implements the sql.Scanner interface (to read from db fields). This makes
// it possible to write to and from the "tags" field in sql drivers
func (o *OutputConfig) Scan(src interface{}) error {
	val, ok := src.([]byte)
	if !ok {
		return errors.New("cant scan anything but bytes")
	}
	err := json.Unmarshal(val, &o)
	return err
}

// Value implements the driver.Valuer interface (for writing to db fields)
func (o OutputConfig) Value() (driver.Value, error) {
	return json.Marshal(o)
}

// Output is data streams from outputs. Note that the CollectionFieldMask is not a
// field on the output table but for simplicity's sake this is retrieved with the
// rest of the output object.
type Output struct {
	ID                  OutputKey
	Type                string
	Config              OutputConfig
	CollectionID        CollectionKey
	Enabled             bool
	CollectionFieldMask FieldMask
	Tags
}

// NewOutput creates a new Output instance
func NewOutput() Output {
	return Output{Config: make(OutputConfig), Tags: NewTags()}
}

// OutputLogEntry is the log entries received from the outputs. Each output has
// a circular buffer with the 10 latest messages from the output. If there's
// multiple log messages with the same message repeated it will just have the
// Repeated field increased. Log entries are used by the end used to diagnose
// the output so the error messages should make sense for end users.
type OutputLogEntry struct {
	Message  string
	Time     time.Time
	Repeated uint8
}

// OutputStatus is used to report the internal state of the forwarder.
type OutputStatus struct {
	Forwarded   int
	Received    int
	ErrorCount  int
	Retransmits int
}
