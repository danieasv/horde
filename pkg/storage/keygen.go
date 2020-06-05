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
	"fmt"
	"math/rand"
	"time"

	"github.com/ExploratoryEngineering/logging"
)

// The SequenceStore keeps track of allocated sequences
type SequenceStore interface {
	// AllocateSequence updates the named sequence with a new value. Returns true on
	// success, false otherwise. If the sequence doesn't exist it will create a
	// a new sequence with the initial value set to `new`.
	AllocateSequence(identifier string, current uint64, new uint64) bool

	// CurrentSequence returns the current value of the sequence. If the sequence
	// doesn't exist it will return 0.
	CurrentSequence(identifier string) (uint64, error)
}

// DefaultSequenceAllocation is the number of identifiers allocated in the
// back end store by the key generator.
const DefaultSequenceAllocation = 10

const maxQueueSize = 10
const bitsForSequence = 46
const bitsForWorker = 14 // 0-16383

// This is implicit: bitsForDC = 4 (ie 0-15)

// KeyGenerator generates identifiers
type KeyGenerator struct {
	name      string
	dcID      uint8
	workerID  uint16
	store     SequenceStore
	generated chan uint64
	request   chan bool
	current   uint64
	max       uint64
}

// NewKeyGenerator creates a new key generator instance
func NewKeyGenerator(dataCenterID uint8, workerID uint16, name string, store SequenceStore) *KeyGenerator {
	return &KeyGenerator{
		fmt.Sprintf("%d/%d/%s", dataCenterID, workerID, name),
		dataCenterID, workerID,
		store, make(chan uint64),
		make(chan bool, maxQueueSize),
		0, 0}
}

// allocateNewBlock allocates a new block of identifiers
func (k *KeyGenerator) allocateNewBlock() {
	value, err := k.store.CurrentSequence(k.name)
	if err != nil && err != ErrNotFound {
		logging.Warning("Unable to retrieve current sequence: %v", err)
	}
	for !k.store.AllocateSequence(k.name, value, value+DefaultSequenceAllocation) {
		value, err = k.store.CurrentSequence(k.name)
		if err != nil {
			logging.Warning("Unable to allocate new block of sequences: %v", err)
			time.Sleep(time.Duration(rand.Intn(10000)) * time.Microsecond)
		}
	}
	k.current = value
	k.max = value + DefaultSequenceAllocation
}

// Start launches the key generator
func (k *KeyGenerator) Start() {
	go func() {
		for range k.request {
			if k.current >= k.max {
				k.allocateNewBlock()
			}
			k.generated <- k.current
			k.current++
		}
	}()
}

// NewID generates a new ID by masking together the current counter, DC id and worker ID
func (k *KeyGenerator) NewID() uint64 {
	k.request <- true
	counter := <-k.generated
	return uint64(k.dcID)<<(bitsForWorker+bitsForSequence) | uint64(k.workerID)<<bitsForSequence | counter
}
