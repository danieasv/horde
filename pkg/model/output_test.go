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
	"reflect"
	"testing"
)

func TestOutput(t *testing.T) {
	op := NewOutput()
	op.ID, _ = NewOutputKeyFromString("0")
	if _, err := NewOutputKeyFromString(op.ID.String()); err != nil {
		t.Fatal("Conversion from String() should work")
	}

	bytes, _ := json.Marshal(map[string]interface{}{"type": "someConfig"})
	if err := op.Config.Scan(bytes); err != nil {
		t.Fatal("Got error scanning bytes: ", err)
	}

	op.Config.Value()

	if err := op.Config.Scan("foo"); err == nil {
		t.Fatal("Expected error with nil scan")
	}
}

func TestOutputConfigNilValue(t *testing.T) {
	cfg := OutputConfig{"abc": nil}
	// As long as this doesn't panic we're OK.
	cfg.HasParameterOfType("abc", reflect.String)
}
