package storage

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
import "github.com/eesrc/horde/pkg/model"

// DownstreamStore is a message store for messages scheduled for downstream
// transport. The messages are added to the store when they are sent and retrieved
// based on the message ID. The oldest messages are retrieved first. Messages are
// identified based on APN, message ID and transport.
// Note: Since the message format is still in flux it's a lot easier to store the
// message as a byte buffer. This isn't ideal wrt digging around but it's
// convenient.
// Note II: This behaves similar to the backlog store but the scope is for all
// (or a subset) of the APNs, not just one APN+NAS.
type DownstreamStore interface {
	// NewMessageID creates a new message ID. Messages created in a
	NewMessageID() model.MessageKey

	// Create creates a new downstream message. The message can be scheduled for
	// a particular APN (if it's a push message) or by device ID (if the APN is
	// "whatever")
	Create(apnID int, nasID int, deviceID model.DeviceKey, id model.MessageKey, transport model.MessageTransport, message []byte) error

	// Retrieve retrieves (push) messages that are scheduled for a particular
	// device.
	Retrieve(apnID int, nasID int, transport model.MessageTransport) (model.MessageKey, []byte, error)

	// Release marks the message as not delivered. When Release is called the
	// will be returned by Retreive some time in the future.
	Release(id model.MessageKey)

	// RetrieveByDevice returns any pending message for a device. The newest message is returned first, then the second newest.
	RetrieveByDevice(deviceID model.DeviceKey, transport model.MessageTransport) (model.MessageKey, []byte, error)

	// Delete removes a downstream message (and implicitly delivered)
	Delete(id model.MessageKey) error
}
