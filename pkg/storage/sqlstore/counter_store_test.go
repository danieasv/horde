package sqlstore

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
	"os"
	"testing"

	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage/storetest"
)

func TestInvalidCounterStoreDriver(t *testing.T) {
	if _, err := NewCounterStore(Parameters{
		Type:             "unknown",
		ConnectionString: "nada",
	}); err == nil {
		t.Fatal("Expected error with unknown driver")
	}
}
func TestEmptyCounterStore(t *testing.T) {
	const dbFile = "counters.db"
	defer os.Remove(dbFile)
	os.Remove(dbFile)

	cs, err := NewCounterStore(Parameters{
		Type:             "sqlite3",
		ConnectionString: dbFile,
	})
	if err != nil {
		t.Fatalf("Expected no errors but got %v", err)
	}
	// Create first without any tables => errors all the way

	if cs.Users() != 0 || cs.Collections() != 0 || cs.Devices() != 0 || cs.Outputs() != 0 || cs.Teams() != 0 {
		t.Fatal("Expected 0 returned for all counters")
	}
}

func TestCounterStore(t *testing.T) {
	const dbFile = "counters.db"
	defer os.Remove(dbFile)
	os.Remove(dbFile)
	// cheat a bit and create a regular sql store with a test environment for the counters
	store, err := NewSQLStore("sqlite3", dbFile, true, 1, 1)
	if err != nil {
		t.Fatal(err)
	}

	env := storetest.NewTestEnvironment(t, store)

	cs, err := NewCounterStore(Parameters{
		Type:             "sqlite3",
		ConnectionString: dbFile})

	if err != nil {
		t.Fatal(err)
	}

	op := model.NewOutput()
	op.ID = store.NewOutputID()
	store.CreateOutput(env.U1.ID, op)

	d := model.NewDevice()
	d.CollectionID = env.C1.ID
	d.ID = store.NewDeviceID()
	d.IMSI = 1
	d.IMEI = 1
	store.CreateDevice(env.U1.ID, d)
	if cs.Collections() == 0 || cs.Devices() == 0 || cs.Teams() == 0 || cs.Users() == 0 {
		t.Fatal("Did not expect 0 for any counters")
	}
}
