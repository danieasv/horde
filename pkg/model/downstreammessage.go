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

// MessageKey is an identifier for messages
type MessageKey storageKey

// NewMessageKeyFromString creates a new MessageKey from a string
func NewMessageKeyFromString(id string) (MessageKey, error) {
	k, err := newKeyFromString(id)
	return MessageKey(k), err
}

// String returns the string representation of the message ID
func (m MessageKey) String() string {
	return storageKey(m).String()
}

// DownstreamMessage is a downstream message; ie a message that should be sent
// to devices.
type DownstreamMessage struct {
	ID        MessageKey       // ID of message
	ApnID     int              // APN ID (may be invalid if the message is cached by the core service)
	NasID     int              // APN ID
	IMSI      int64            // IMSI for destination device
	Transport MessageTransport // Transport to use
	Port      int              // Port number to use (UDP or CoAP)
	Path      string           // Path (CoAP)
	Payload   []byte           // Payload of message
}
