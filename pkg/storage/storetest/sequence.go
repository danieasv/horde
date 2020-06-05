package storetest

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
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/eesrc/horde/pkg/storage"
)

// SequenceTest runs tests on a SequenceStore implementation. For unit testing only.
func SequenceTest(t *testing.T, s1 storage.SequenceStore) {

	kg1a := storage.NewKeyGenerator(1, 1, "one", s1)
	kg1a.Start()
	kg1b := storage.NewKeyGenerator(2, 1, "two", s1)
	kg1b.Start()
	kg1c := storage.NewKeyGenerator(1, 1, "one", s1)
	kg1c.Start()
	kg2 := storage.NewKeyGenerator(2, 1, "two", s1)
	kg2.Start()

	generatedIDs := make(map[uint64]bool)
	generated := 0

	for i := 0; i < 100; i++ {
		generatedIDs[kg1a.NewID()] = true
		generated++
		generatedIDs[kg1b.NewID()] = true
		generated++
		generatedIDs[kg1c.NewID()] = true
		generated++
		generatedIDs[kg2.NewID()] = true
		generated++
	}
	if generated != len(generatedIDs) {
		t.Fatalf("Duplicate IDs. Generated: %d, actual: %d", generated, len(generatedIDs))
	}

	// Test with two sequences in parallel
	const id = "sequence-test"
	wg := &sync.WaitGroup{}
	wg.Add(2)

	errs := make(chan error, 3000)
	// Make two sequences and run in parallell.
	test := func(s storage.SequenceStore, errs chan error) {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			var err error
			fails := 0
			current, err := s.CurrentSequence(id)
			if err != nil && err != storage.ErrNotFound {
				errs <- fmt.Errorf("unable to retrieve current: %v", err)
				return
			}
			for !s.AllocateSequence(id, current, current+100) {
				time.Sleep(time.Duration(rand.Intn(100)) * time.Microsecond)
				current, err = s.CurrentSequence(id)
				if err != nil {
					errs <- fmt.Errorf("unable to read current value from sequenec: %v", err)
					return
				}
				fails++
				if fails > 100 {
					errs <- errors.New("unable to allocate sequence")
					return
				}
			}
		}
	}
	go test(s1, errs)
	go test(s1, errs)

	wg.Wait()
	select {
	case err := <-errs:
		t.Fatal(err)
	default:
	}

}
