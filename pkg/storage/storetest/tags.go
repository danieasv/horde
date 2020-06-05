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
	"testing"

	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
)

// TagRetrieveFunc is a function to retrieve tags on an entity
type tagRetrieveFunc func(userid model.UserKey, id string) (model.Tags, error)

// TagUpdateFunc is a function to update tags on an entity
type tagUpdateFunc func(userid model.UserKey, id string, tags model.Tags) error

// Test the set and update functions for tags. The supplied key is for
// an entity that U1 has admin access to and U2 has member access to in the test environment.
// If u2CanGet is false, then U2 has no access to the entity.
func testTagSetAndGet(t *testing.T, env TestEnvironment, key string, u2CanGet bool, setTag tagUpdateFunc, getTag tagRetrieveFunc) {
	tags, err := getTag(env.U1.ID, key)
	if err != nil {
		t.Fatal("Unable to retrieve tags: ", err)
	}

	tags.SetTag("test1", "foo")
	tags.SetTag("test2", "bar")
	tags.SetTag("test3", "baz")

	if err := setTag(env.U1.ID, key, tags); err != nil {
		t.Fatal("Unable to update tags: ", err)
	}

	if _, err := getTag(env.U1.ID, "foobar"); err == nil {
		t.Fatal("Expected error when using bogus key")
	}
	if err := setTag(env.U1.ID, "foobarbaz", tags); err == nil {
		t.Fatal("Expected error when using bogus key")
	}

	u2getErr := error(nil)
	u2setErr := storage.ErrAccess
	if !u2CanGet {
		u2getErr = storage.ErrNotFound
		u2setErr = storage.ErrNotFound
	}
	if _, err := getTag(env.U2.ID, key); err != u2getErr {
		t.Fatal("Unexpected error for user 2 access: ", err)
	}
	if err := setTag(env.U2.ID, key, tags); err != u2setErr {
		t.Fatal("Should not be allowed to update tag with only member access. err = ", err)
	}

	if err := setTag(env.U3.ID, key, tags); err != storage.ErrNotFound {
		t.Fatal("Should not be allowed to update tag I have no access to. err = ", err)
	}
	if _, err := getTag(env.U3.ID, key); err != storage.ErrNotFound {
		t.Fatal("Should not be able to retrieve tags I have no access to. err = ", err)
	}

	// Assumption: Nothing gets the ID 0. I might be wrong. I might be the one
	// thinking "what an idiot"
	if _, err := getTag(env.U1.ID, "0"); err != storage.ErrNotFound {
		t.Fatal("Should not be able to retrieve tags with unknown ID. err = ", err)
	}
	if err := setTag(env.U1.ID, "0", tags); err != storage.ErrNotFound {
		t.Fatal("Should not be able to update tags with unknown ID. err = ", err)
	}
}
