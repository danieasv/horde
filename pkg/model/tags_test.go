package model
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
	"encoding/json"
	"testing"
)

func TestTags(t *testing.T) {
	tags := NewTags()

	if tags.TagExists("foo") {
		t.Fatal("foo should not exist")
	}

	tags.SetTag(" bar ", "baz  ")

	if !tags.TagExists("bar") {
		t.Fatal("bar should exist")
	}
	if tags.GetTag("bar") != "baz" {
		t.Fatal("bar does not have the correct value")
	}

	tags.SetTag("BAR", "")
	if tags.TagExists("bar") {
		t.Fatal("bar should no longer exist")
	}

	if tags.IsValidTag("", "") {
		t.Fatal("Empty keys and values shouldn't be valid")
	}
	if !tags.IsValidTag("alert('this is valid')", "") {
		t.Fatal("Everyone likes javascript")
	}

	tags.SetTag("foo", "bar")
	vals := tags.TagData()
	delete(vals, "foo")
	vals["bar"] = "foo"
	tags.SetTags(vals)

	bytes, _ := json.Marshal(vals)
	tags.TagMap.Scan(bytes)
	tags.TagMap.Value()

	if tags.TagMap.Scan("foo") == nil {
		t.Fatal("Expected error when scanning a string")
	}
}
