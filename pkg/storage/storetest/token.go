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

	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
)

// TestTokenStore executes unit tests on a token store implementation
func testTokenStore(env TestEnvironment, store storage.DataStore, t *testing.T) {
	t1 := model.NewToken()
	t1.UserID = env.U1.ID
	t1.GenerateToken()
	t1.Resource = "/one"
	t1.Write = true

	if err := store.CreateToken(t1); err != nil {
		t.Fatal("Coulnd't create token 1")
	}

	list, err := store.ListTokens(env.U1.ID)
	if err != nil {
		t.Fatal("Unable to retrieve token list for u1: ", err)
	}
	if len(list) != 1 {
		t.Fatalf("Expected only one token returned for u1 (%s) but got %d tokens (%+v)", env.U1.ID, len(list), list)
	}

	list, err = store.ListTokens(env.U2.ID)
	if err != nil {
		t.Fatal("Unable to retrieve token list for u2: ", err)
	}
	if len(list) != 0 {
		t.Fatal("Expected zero tokens returned for u2 but got ", len(list))
	}

	t2 := model.NewToken()
	t2.UserID = env.U2.ID
	t2.GenerateToken()
	t2.Resource = "/two"

	if err := store.CreateToken(t2); err != nil {
		t.Fatal("Couldn't create token 2")
	}
	if err := store.CreateToken(t2); err != storage.ErrAlreadyExists {
		t.Fatal("Shouldn't be able to create the same token twice. Erroro: ", err)
	}

	tx := model.NewToken()
	tx.GenerateToken()
	tx.Resource = "/"
	tx.Write = true

	t2.Resource = "/newtwo"
	t2.SetTag("name", "number2")
	if err := store.UpdateToken(t2); err != nil {
		t.Fatal("Unable to update token")
	}

	if err := store.UpdateToken(tx); err != storage.ErrNotFound {
		t.Fatal("Should not be able to update a token that doesn't exist but got ", err)
	}

	tokens, err := store.ListTokens(t2.UserID)
	if err != nil {
		t.Fatal("Error listing tokens: ", err)
	}
	for _, v := range tokens {
		if v.Token == t2.Token {
			if !reflect.DeepEqual(v, t2) || v.GetTag("name") != "number2" {
				t.Fatalf("Returned token does not match (%+v != %+v)", v, t2)
			}
		}
	}

	// Test tag implementations
	testTagSetAndGet(t, env, t1.Token, false, store.UpdateTokenTags, store.RetrieveTokenTags)

	if err := store.DeleteToken(env.U1.ID, t1.Token); err != nil {
		t.Fatal("Unable to remove token 1")
	}
	if err := store.DeleteToken(env.U1.ID, t2.Token); err != storage.ErrNotFound {
		t.Fatal("Should not be able to remove someone else's token but got ", err)
	}
	store.DeleteToken(env.U2.ID, t2.Token)
	if err := store.DeleteToken(env.U2.ID, t2.Token); err != storage.ErrNotFound {
		t.Fatal("Should not be able to remove the same token twice")
	}
}
