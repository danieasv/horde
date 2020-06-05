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
	"sync"
	"time"

	"github.com/ExploratoryEngineering/logging"
	"github.com/ExploratoryEngineering/pubsub"
	"github.com/eesrc/horde/pkg/model"
)

// queueLength is the length of the event router's queue
const queueLength = 100

type outputEntry struct {
	ch     <-chan interface{}
	output Output
}

// localManager is a manager running on the local instance. It will only keep
// track of outputs launched locally.
type localManager struct {
	running   map[model.OutputKey]outputEntry
	publisher pubsub.EventRouter
	mutex     *sync.Mutex
}

// NewLocalManager creates a new manager running locally
func NewLocalManager() Manager {
	listOutputTypes()
	return &localManager{
		running:   make(map[model.OutputKey]outputEntry),
		publisher: pubsub.NewEventRouter(queueLength),
		mutex:     &sync.Mutex{},
	}
}

func (l *localManager) Verify(output model.Output) (model.ErrorMessage, error) {
	op, err := NewOutput(output.Type)
	if err != nil {
		return model.ErrorMessage{"type": err.Error()}, err
	}
	return op.Validate(output.Config)
}

func (l *localManager) Refresh(outputs []model.Output, systemFieldMask model.FieldMask) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	for _, v := range outputs {
		_, exists := l.running[v.ID]
		if exists {
			continue
		}
		if !v.Enabled {
			continue
		}
		new, err := NewOutput(v.Type)
		if err != nil {
			logging.Warning("Unable to launch output with ID %v: %v. Ignoring", v.ID, err)
			continue
		}
		ch := l.Subscribe(v.CollectionID)
		new.Start(v.Config, v.CollectionFieldMask, systemFieldMask, ch)
		l.running[v.ID] = outputEntry{ch: ch, output: new}
	}
}

func (l *localManager) Update(output model.Output, systemFieldMask model.FieldMask) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	v, exists := l.running[output.ID]
	if exists {
		l.Unsubscribe(v.ch)
		v.output.Stop(stopTimeout)
	}

	newOutput, err := NewOutput(output.Type)
	if err != nil {
		logging.Warning("Couldn't create new output: %v", err)
		return err
	}

	if !output.Enabled {
		logging.Info("Won't start disabled output with ID %s (type: %s)", output.ID.String(), output.Type)
		return nil
	}
	ch := l.Subscribe(output.CollectionID)
	newOutput.Start(output.Config, output.CollectionFieldMask, systemFieldMask, ch)
	l.running[output.ID] = outputEntry{ch: ch, output: newOutput}
	return nil
}

func (l *localManager) Stop(key model.OutputKey) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	v, exists := l.running[key]
	if !exists {
		return errors.New("unknown output")
	}
	delete(l.running, key)
	l.Unsubscribe(v.ch)
	v.output.Stop(stopTimeout)
	return nil
}

const stopTimeout = 3 * time.Second

func (l *localManager) Shutdown() {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	for k, v := range l.running {
		l.Unsubscribe(v.ch)
		v.output.Stop(stopTimeout)
		delete(l.running, k)
	}
}

func (l *localManager) Get(key model.OutputKey) (Output, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	ret, exists := l.running[key]
	if !exists {
		return nil, errors.New("unknown output")
	}
	return ret.output, nil
}

func (l *localManager) Publish(msg model.DataMessage) {
	l.publisher.Publish(msg.Device.CollectionID, msg)
}

func (l *localManager) Subscribe(collectionID model.CollectionKey) <-chan interface{} {
	return l.publisher.Subscribe(collectionID)
}

func (l *localManager) Unsubscribe(ch <-chan interface{}) {
	l.publisher.Unsubscribe(ch)
}
