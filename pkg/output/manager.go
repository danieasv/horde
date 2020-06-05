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

	"github.com/ExploratoryEngineering/pubsub"
	"github.com/eesrc/horde/pkg/model"
)

// Manager is responsible for keeping track of outputs and the mapping
// between the storage outputs and the actual running outputs.
type Manager interface {

	// Verify checks the output's configuration without launching it
	Verify(model.Output) (model.ErrorMessage, error)

	// Load loads outputs from backend store and launches the ones that aren't
	// up and running yet. The Load call might be performed multiple times to
	// update the list. The field mask is the system-level field mask, ie forced
	// field mask
	Refresh([]model.Output, model.FieldMask)

	// Update refreshes the output. If it isn't launched yet it will be
	// launched. If it is already running the new configuration will be
	// applied. The field mask is the system field mask, ie the forced
	// field mask
	Update(model.Output, model.FieldMask) error

	// Stop stops a single output, typically if they have been deleted.
	Stop(model.OutputKey) error

	// Shutdown shuts down all of the running outputs.
	Shutdown()

	// Get returns the output. If the output isn't running or is unknown it
	// will return an error. The output includes the current logs.
	Get(model.OutputKey) (Output, error)

	// Publish publishes a data message to the running outputs. If there's no
	// outputs subscribing to the data it will be discarded.
	Publish(model.DataMessage)

	// Subscribe subscribes to a topic
	Subscribe(collectionID model.CollectionKey) <-chan interface{}

	// Unsubscribe unsubscribes from a topic
	Unsubscribe(ch <-chan interface{})
}

// NewManager creates a new manager for outputs.
func NewManager() Manager {
	return nil
}

type dummyManager struct {
	router pubsub.EventRouter
}

func (m *dummyManager) Refresh([]model.Output, model.FieldMask) {

}

func (m *dummyManager) Update(model.Output, model.FieldMask) error {
	return nil
}

func (m *dummyManager) Stop(model.OutputKey) error {
	return nil
}

func (m *dummyManager) Shutdown() {
	// Nothing
}

func (m *dummyManager) Get(id model.OutputKey) (Output, error) {
	return nil, errors.New("not implemented")
}

func (m *dummyManager) Verify(config model.Output) (model.ErrorMessage, error) {
	op, err := NewOutput(config.Type)
	if err != nil {
		return nil, err
	}
	return op.Validate(config.Config)
}

func (m *dummyManager) Publish(msg model.DataMessage) {
	m.router.Publish(msg.Device.CollectionID, msg)
}

func (m *dummyManager) Subscribe(collectionID model.CollectionKey) <-chan interface{} {
	return m.router.Subscribe(collectionID)
}

func (m *dummyManager) Unsubscribe(ch <-chan interface{}) {
	m.router.Unsubscribe(ch)
}

// NewDummyManager For testing: Return a dummy manager
func NewDummyManager() Manager {
	return &dummyManager{router: pubsub.NewEventRouter(2)}
}
