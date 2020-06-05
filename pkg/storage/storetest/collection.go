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
	"reflect"
	"testing"
	"time"

	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
)

// TestCollectionStore tests a svc.CollectionStore implementation
func testCollectionStore(env TestEnvironment, s storage.DataStore, t *testing.T) {
	for i := 0; i < 100; i++ {
		id1 := s.NewCollectionID()
		id2 := s.NewCollectionID()
		if id1 == id2 {
			t.Fatal("CollectionNewID generated identical IDs")
		}
	}

	{
		c := model.NewCollection()
		c.ID = s.NewCollectionID()
		c.TeamID = env.T21.ID
		c.SetTag("name", "nobody home")
		if err := s.CreateCollection(env.U1.ID, c); err != storage.ErrAccess {
			t.Fatal("Should not be able to create collection in a team where I'm not admin: ", err)
		}
	}

	// A private collection for U1
	c1 := model.NewCollection()
	c1.ID = s.NewCollectionID()
	c1.TeamID = env.T1.ID
	c1.SetTag("name", "test collection 1")
	if err := s.CreateCollection(env.U1.ID, c1); err != nil {
		t.Fatal("Unable to create collection for U1: ", err)
	}

	// A private collection for U2
	c2 := model.NewCollection()
	c2.ID = s.NewCollectionID()
	c2.TeamID = env.T2.ID
	c2.SetTag("name", "test collection 2")
	if err := s.CreateCollection(env.U2.ID, c2); err != nil {
		t.Fatal("Unable to create collection for U2: ", err)
	}

	if err := s.CreateCollection(env.U1.ID, c1); err != storage.ErrAlreadyExists {
		t.Fatal("Should not be able to create the same collection twice. err = ", err)
	}

	// Collections for U1 + U2/U2 + U1
	c12 := model.NewCollection()
	c12.ID = s.NewCollectionID()
	c12.TeamID = env.T12.ID
	c12.SetTag("name", "test collection 12")
	if err := s.CreateCollection(env.U1.ID, c12); err != nil {
		t.Fatalf("Unable to create collection c12: %v", err)
	}

	c21 := model.NewCollection()
	c21.ID = s.NewCollectionID()
	c21.TeamID = env.T21.ID
	c21.SetTag("name", "test collection 21")
	if err := s.CreateCollection(env.U2.ID, c21); err != nil {
		t.Fatalf("unable to create collection c21: %v", err)
	}

	// Another collection
	c3 := model.NewCollection()
	c3.ID = s.NewCollectionID()
	c3.TeamID = env.T3.ID
	c3.SetTag("name", "test collection 3")
	if err := s.CreateCollection(env.U3.ID, c3); err != nil {
		t.Fatalf("Unable to create collection c3: %v", err)
	}

	hasElements := func(arr []model.Collection, ids ...model.CollectionKey) bool {
		if arr == nil || len(arr) < len(ids) {
			return false
		}
		found := 0
		for _, v := range ids {
			for i := range arr {
				if arr[i].ID == v {
					found++
					break
				}
			}
		}
		return found == len(ids)
	}

	// Retrieve collections for U1; should contain c1, c12, c21
	list1, err := s.ListCollections(env.U1.ID)
	if err != nil {
		t.Fatal("Could not retrieve collections for U1: ", err)
	}
	if !hasElements(list1, c1.ID, c12.ID, c21.ID) {
		t.Fatalf("Missing one or more elements (%s, %s, %s), from list 1. List = %+v", c1.ID, c12.ID, c21.ID, list1)
	}

	// ...while U2 should see C2 + C12 + C21
	list2, err := s.ListCollections(env.U2.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !hasElements(list2, c2.ID, c21.ID, c12.ID) {
		t.Fatalf("Missing elements from list 2. List = %+v", list2)
	}

	rc1, err := s.RetrieveCollection(env.U1.ID, c1.ID)
	if err != nil {
		t.Fatal("Unable to retrieve collection 1: ", err)
	}
	if !reflect.DeepEqual(rc1, c1) {
		t.Fatalf("Retrived collection different from created collection: %+v != %+v", c1, rc1)
	}

	rc2, err := s.RetrieveCollection(env.U1.ID, c21.ID)
	if err != nil {
		t.Fatal("Should be able to retrieve collection 21 as user 1: ", err)
	}
	if !reflect.DeepEqual(rc2, c21) {
		t.Fatalf("Retrieved collection 2 different from created: %+v != %+v", c21, rc2)
	}

	if _, err := s.RetrieveCollection(env.U2.ID, c3.ID); err != storage.ErrNotFound {
		t.Fatalf("(u2 -> c3) Expected not found error but got %+v", err)
	}
	if _, err := s.RetrieveCollection(env.U2.ID, c1.ID); err != storage.ErrNotFound {
		t.Fatalf("(u2 -> c1) Exepcted not found error but got %+v", err)
	}
	if _, err := s.RetrieveCollection(env.U1.ID, s.NewCollectionID()); err != storage.ErrNotFound {
		t.Fatal("(unknonw) Expected not found error for unknown collection")
	}

	// Modify c1, c12, c21 and attempt updates as user U1
	c1.SetTag("updated", "true")
	c12.SetTag("updated", "true")
	c21.SetTag("updated", "true")
	if err := s.UpdateCollection(env.U1.ID, c1); err != nil {
		t.Fatal("(u1->c1) Did not expect update error: ", err)
	}
	if err := s.UpdateCollection(env.U1.ID, c12); err != nil {
		t.Fatal("(u1->c12) Did not expect update error: ", err)
	}
	if err := s.UpdateCollection(env.U1.ID, c21); err != storage.ErrAccess {
		t.Fatal("(u1->c21) Expected update error. err = ", err)
	}
	if err := s.UpdateCollection(env.U1.ID, env.C3); err != storage.ErrNotFound {
		t.Fatal("(c3) Expected update error. err = ", err)
	}
	unknown := model.NewCollection()
	unknown.ID = s.NewCollectionID()
	unknown.TeamID = env.T1.ID
	if err := s.UpdateCollection(env.U1.ID, unknown); err != storage.ErrNotFound {
		t.Fatal("(unknown) Expected update error. err = ", err)
	}
	// Moving collection between teams t1 -> t12 should work, t1->t21 should not
	c1.TeamID = env.T21.ID
	if err := s.UpdateCollection(env.U1.ID, c1); err == nil {
		t.Fatal("c: t1->t21 Expected update error")
	}
	c1.TeamID = env.T12.ID
	if err := s.UpdateCollection(env.U1.ID, c1); err != nil {
		t.Fatal("c: t1->t12 Did not expect update error: ", err)
	}
	c1.TeamID = env.T1.ID
	if err := s.UpdateCollection(env.U1.ID, c1); err != nil {
		t.Fatal("Could not set the team if for C1: ", err)
	}
	testCollectionFirmware(t, env, s)

	testTagSetAndGet(t, env, c12.ID.String(), true, s.UpdateCollectionTags, s.RetrieveCollectionTags)

	if err := s.DeleteCollection(env.U1.ID, c1.ID); err != nil {
		t.Fatal("u1 rm c1: unexpected error: ", err)
	}
	if err := s.DeleteCollection(env.U1.ID, c1.ID); err != storage.ErrNotFound {
		t.Fatal("Expected not found error when deleting collection twice")
	}
	if err := s.DeleteCollection(env.U1.ID, c12.ID); err != nil {
		t.Fatal("u1 rm c12: unexpected error: ", err)
	}
	if err := s.DeleteCollection(env.U1.ID, c21.ID); err != storage.ErrAccess {
		t.Fatal("u1 rm c21: unexpected error: ", err)
	}
	if err := s.DeleteCollection(env.U2.ID, c2.ID); err != nil {
		t.Fatal("u2 rm c2: unexpected error: ", err)
	}
	if err := s.DeleteCollection(env.U2.ID, c21.ID); err != nil {
		t.Fatal("u2 rm c21: unexpected error: ", err)
	}
	if err := s.DeleteCollection(env.U1.ID, c3.ID); err != storage.ErrNotFound {
		t.Fatal("u1 rm c3: unexpected error: ", err)
	}

	if err := s.DeleteCollection(env.U1.ID, unknown.ID); err != storage.ErrNotFound {
		t.Fatal("Expected error for unknown collection. err = ", err)
	}
}

func testCollectionFirmware(t *testing.T, e TestEnvironment, s storage.DataStore) {
	fw11 := model.NewFirmware()
	fw11.ID = s.NewFirmwareID()
	fw11.Version = "1.0"
	fw11.Filename = "file1.0"
	fw11.Length = 100
	fw11.SHA256 = "aabb"
	fw11.Created = time.Now()
	fw11.CollectionID = e.C1.ID

	if err := s.CreateFirmware(e.U1.ID, fw11); err != nil {
		t.Fatal("Unable to create firmware entity: ", err)
	}

	fw12 := model.NewFirmware()
	fw12.ID = s.NewFirmwareID()
	fw12.Version = "2.0"
	fw12.Filename = "file2.0"
	fw12.Length = 200
	fw12.SHA256 = "ccdd"
	fw12.Created = time.Now()
	fw12.CollectionID = e.C1.ID
	if err := s.CreateFirmware(e.U1.ID, fw12); err != nil {
		t.Fatal("Unable to create firmware entity: ", err)
	}

	fw2 := model.NewFirmware()
	fw2.ID = s.NewFirmwareID()
	fw2.Version = "3.0"
	fw2.Filename = "file3.0"
	fw2.Length = 300
	fw2.SHA256 = "eeff"
	fw2.Created = time.Now()
	fw2.CollectionID = e.C2.ID
	if err := s.CreateFirmware(e.U2.ID, fw2); err != nil {
		t.Fatal("Unable to create firmware entity: ", err)
	}

	e.C1.Firmware.CurrentFirmwareID = fw11.ID
	e.C1.Firmware.TargetFirmwareID = fw12.ID
	if err := s.UpdateCollection(e.U1.ID, e.C1); err != nil {
		t.Fatal("Unable to update collection: ", err)
	}

	c, err := s.RetrieveCollection(e.U1.ID, e.C1.ID)
	if err != nil {
		t.Fatal("Unable to retrieve collection")
	}
	if c.Firmware.CurrentFirmwareID != fw11.ID || c.Firmware.TargetFirmwareID != fw12.ID {
		t.Fatalf("Firmware isn't updated on collection: %+v != %+v", e.C1, c)
	}

	c.Firmware.TargetFirmwareID = fw2.ID
	if err := s.UpdateCollection(e.U1.ID, c); err != storage.ErrAccess {
		t.Fatal("Expected access error but got ", err)
	}
	// Clean up store afterwards
	e.C1.Firmware.CurrentFirmwareID = 0
	e.C1.Firmware.TargetFirmwareID = 0
	if err := s.UpdateCollection(e.U1.ID, e.C1); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteFirmware(e.U1.ID, e.C1.ID, fw11.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteFirmware(e.U1.ID, e.C1.ID, fw12.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteFirmware(e.U2.ID, e.C2.ID, fw2.ID); err != nil {
		t.Fatal(err)
	}
}
