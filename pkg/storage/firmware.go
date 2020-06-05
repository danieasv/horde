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
import (
	"io"

	"github.com/eesrc/horde/pkg/model"
)

// The FirmwareImageStore is the storage for the actual firmware images. The
//
type FirmwareImageStore interface {
	// Create persists a firmware image in the backend store. The SHA256 checksum
	// is returned
	Create(model.FirmwareKey, io.Reader) (string, error)
	// Retrieve retrieves a firmware image from the backend store. The reader should
	// be closed when the client has finished reading the data.
	Retrieve(model.FirmwareKey) (io.ReadCloser, error)
	// Delete removes the firmware image from the backend store.
	Delete(model.FirmwareKey) error
}
