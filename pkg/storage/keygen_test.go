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
	"errors"
	"testing"
)

type testStore struct {
	sequences map[string]uint64
	errors    int
}

func (k *testStore) AllocateSequence(identifier string, current uint64, new uint64) bool {
	if k.errors > 0 {
		k.errors--
		return false
	}
	v, ok := k.sequences[identifier]
	if ok && v != current {
		return false
	}
	k.sequences[identifier] = new
	return true
}
func (k *testStore) CurrentSequence(identifier string) (uint64, error) {
	if k.errors > 0 {
		k.errors--
		return 0, errors.New("got error")
	}
	v, ok := k.sequences[identifier]
	if !ok {
		// There's nothing with that identifier but it's OK.
		return 0, nil
	}

	return v, nil
}

func TestKeyGenerator(t *testing.T) {
	store := &testStore{sequences: make(map[string]uint64)}
	kg1 := NewKeyGenerator(1, 2, "test", store)
	kg2 := NewKeyGenerator(1, 3, "test", store)
	kg1.Start()
	kg2.Start()

	var ids []uint64
	for i := 0; i < 10; i++ {
		ids = append(ids, kg1.NewID())
		ids = append(ids, kg2.NewID())
	}

	store.errors = 20
	for i := 0; i < 10; i++ {
		ids = append(ids, kg1.NewID())
		ids = append(ids, kg2.NewID())
	}
	// All IDs should be unique
	for i, a := range ids {
		for j, b := range ids {
			if j == i {
				continue
			}
			if a == b {
				t.Fatalf("Got colliding IDs at index %d and %d: %d", i, j, a)
			}
		}
	}
}
