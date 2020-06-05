package restapi

//
//Copyright 2020 Telenor Digital AS
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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eesrc/horde/pkg/storage/sqlstore"
	"github.com/eesrc/horde/pkg/storage/storetest"

	"github.com/eesrc/horde/pkg/model"
)

func TestTokenHandler(t *testing.T) {
	invocations := 0
	innerHandler := func(w http.ResponseWriter, r *http.Request) {
		invocations++
	}

	tokenStore := sqlstore.NewMemoryStore()
	env := storetest.NewTestEnvironment(t, tokenStore)
	tokens := []model.Token{
		model.Token{Token: "all", UserID: env.U1.ID, Resource: "/", Write: true, Tags: model.NewTags()},
		model.Token{Token: "read", UserID: env.U2.ID, Resource: "/applications", Write: false, Tags: model.NewTags()},
	}
	for _, v := range tokens {
		tokenStore.CreateToken(v)
	}
	s := restServer{}
	tokenHandler := s.createTokenHandler(innerHandler, tokenStore)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "http://example.com/applications", strings.NewReader("hello world"))

	testInvocations := func(expected int) {
		tokenHandler.ServeHTTP(w, r)
		if invocations != expected {
			t.Fatalf("Expected %d invocations but invocations is %d", expected, invocations)
		}
	}

	r.Header.Set(tokenHeader, "all")
	testInvocations(1)

	r.Header.Set(tokenHeader, "read")
	testInvocations(2)

	r.Method = http.MethodPost
	testInvocations(2)

	r.Method = http.MethodPatch
	testInvocations(2)

	r.Method = http.MethodDelete
	testInvocations(2)

	r.Method = http.MethodGet
	r.URL.Path = "/secrets"
	testInvocations(2)

	r.URL.Path = "/applications"

	// No header means pass on to next handler
	r.Header.Set(tokenHeader, "")
	testInvocations(2)

	r.Header.Set(tokenHeader, "nouser")
	testInvocations(2)

}
