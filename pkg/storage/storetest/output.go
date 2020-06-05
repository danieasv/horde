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
	"fmt"
	"testing"

	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
)

// TestOutputStore runs a series of tests on a output store implementation
func testOutputStore(e TestEnvironment, s storage.DataStore, t *testing.T) {
	o1 := make([]model.Output, 0)
	o2 := make([]model.Output, 0)
	o12 := make([]model.Output, 0)
	o21 := make([]model.Output, 0)
	const numOutputs = 10

	for i := 0; i < numOutputs; i++ {
		o := model.NewOutput()
		o.ID = s.NewOutputID()
		o.Type = fmt.Sprintf("Type %d of %d", i+1, numOutputs)
		o.Config["foo"] = "bar"
		o.Config["bar"] = 1
		o.Config["baz"] = true
		o.Tags.SetTag("name", fmt.Sprintf("output %d", i))
		o.CollectionID = e.C1.ID
		if err := s.CreateOutput(e.U1.ID, o); err != nil {
			t.Fatal("Error creating output for U1: ", err)
		}
		o1 = append(o1, o)

		o.ID = s.NewOutputID()
		o.CollectionID = e.C2.ID
		if err := s.CreateOutput(e.U2.ID, o); err != nil {
			t.Fatal("Error creating output for U2: ", err)
		}
		o2 = append(o2, o)

		o.ID = s.NewOutputID()
		o.CollectionID = e.C12.ID
		if err := s.CreateOutput(e.U1.ID, o); err != nil {
			t.Fatal("Unable to create output for C12: ", err)
		}
		o12 = append(o12, o)

		o.ID = s.NewOutputID()
		o.CollectionID = e.C21.ID
		if err := s.CreateOutput(e.U2.ID, o); err != nil {
			t.Fatal("Unable to create output for C21: ", err)
		}
		o21 = append(o21, o)
	}

	if err := s.CreateOutput(e.U2.ID, o2[0]); err == nil {
		t.Fatal("Should not allow multiple outputs with same id")
	}
	// Should not be allowed to create new output for collection
	// I don't own
	o := model.NewOutput()
	o.ID = s.NewOutputID()
	o.CollectionID = e.C21.ID
	o.Type = "foo"
	o.Config["a"] = "b"
	o.Tags.SetTag("name", "meh")
	if err := s.CreateOutput(e.U1.ID, o); err != storage.ErrAccess {
		t.Fatal("Expected access error when creating output in collection and not admin: err = ", err)
	}

	// List outputs. Should get a list for all of the collections I'm a member of
	list, err := s.ListOutputs(e.U1.ID, e.C1.ID)
	if err != nil {
		t.Fatal("Got error retrieving list of outputs")
	}
	if len(list) != len(o1) {
		t.Fatal("List should contain 10 elements")
	}
	list, err = s.ListOutputs(e.U2.ID, e.C12.ID)
	if err != nil {
		t.Fatal("Got error retrieving list of outputs")
	}
	// lists are different but both should be the same length
	if len(list) != len(o12) {
		t.Fatal("Missing elements. Expected ", len(o12), " got ", len(list))
	}

	if list, err := s.ListOutputs(e.U1.ID, e.C2.ID); err == nil {
		t.Fatal("Expected error when retrieving private list. Got none. List = ", list)
	}

	// retrieve outputs. Same story.
	if _, err := s.RetrieveOutput(e.U1.ID, e.C1.ID, o1[0].ID); err != nil {
		t.Fatal("Unable to retrieve output: ", err)
	}
	if _, err := s.RetrieveOutput(e.U1.ID, e.C2.ID, o2[0].ID); err == nil {
		t.Fatal("Expected error for private output")
	}
	if _, err := s.RetrieveOutput(e.U1.ID, e.C1.ID, s.NewOutputID()); err == nil {
		t.Fatal("Expected error for unknown output")
	}

	testTagSetAndGet(t, e, o12[0].ID.String(), true, s.UpdateOutputTags, s.RetrieveOutputTags)

	// Update outputs
	o1[0].CollectionID = e.C12.ID
	if err := s.UpdateOutput(e.U1.ID, e.C1.ID, o1[0]); err != nil {
		t.Fatal("Unable to update item: ", err)
	}
	if err := s.UpdateOutput(e.U1.ID, e.C21.ID, o21[0]); err != storage.ErrAccess {
		t.Fatal("Should not be able to update item I don't own, err =", err)
	}
	if err := s.UpdateOutput(e.U2.ID, e.C12.ID, o21[0]); err != storage.ErrNotFound {
		t.Fatal("Should not be able to update item I don't own, err =", err)
	}
	if err := s.UpdateOutput(e.U1.ID, e.C2.ID, o2[0]); err != storage.ErrNotFound {
		t.Fatal("Should not be able to update item I don't own, err =", err)
	}
	o1[0].CollectionID = e.C2.ID
	if err := s.UpdateOutput(e.U2.ID, e.C2.ID, o1[0]); err != storage.ErrNotFound {
		t.Fatal("Should not be able to update item I don't own, err =", err)
	}

	o1[0].CollectionID = e.C1.ID
	s.UpdateOutput(e.U1.ID, e.C12.ID, o1[0])

	tmp := model.NewOutput()
	tmp.ID = s.NewOutputID()
	tmp.CollectionID = e.C1.ID
	if err := s.UpdateOutput(e.U1.ID, e.C1.ID, tmp); err != storage.ErrNotFound {
		t.Fatal("Should not be able to update unknown output")
	}

	if _, err := s.OutputListAll(); err != nil {
		t.Fatal("Should be able to list all outputs")
	}
	// Delete outputs
	if err := s.DeleteOutput(e.U1.ID, e.C2.ID, o2[0].ID); err != storage.ErrNotFound {
		t.Fatal("Should not be able to delete outputs I don't own, err = ", err)
	}
	if err := s.DeleteOutput(e.U1.ID, e.C1.ID, o2[0].ID); err != storage.ErrNotFound {
		t.Fatal("Should not be able to delete outputs I don't own, err = ", err)
	}
	if err := s.DeleteOutput(e.U1.ID, e.C21.ID, o21[0].ID); err != storage.ErrAccess {
		t.Fatal("Should not be able to delete outputs I don't administer, err = ", err)
	}
	if err := s.DeleteOutput(e.U1.ID, e.C1.ID, s.NewOutputID()); err != storage.ErrNotFound {
		t.Fatal("Should not be allowed to delete unknown outputs")
	}
	for i := 0; i < numOutputs; i++ {
		if err := s.DeleteOutput(e.U1.ID, e.C1.ID, o1[i].ID); err != nil {
			t.Fatal("Should be allowed to delete my own outputs")
		}
		if err := s.DeleteOutput(e.U2.ID, e.C2.ID, o2[i].ID); err != nil {
			t.Fatal("Should be allowed to delete my own outputs")
		}
		if err := s.DeleteOutput(e.U1.ID, e.C12.ID, o12[i].ID); err != nil {
			t.Fatal("Should be allowed to delete my own outputs")
		}
		if err := s.DeleteOutput(e.U2.ID, e.C21.ID, o21[i].ID); err != nil {
			t.Fatal("Should be allowed to delete my own outputs")
		}
	}
}
