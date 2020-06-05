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
	"database/sql/driver"
	"encoding/json"
	"errors"
	"strings"
)

// TagMapData is the tag types
type TagMapData map[string]string

// Scan implements the sql.Scanner interface (to read from db fields). This makes
// it possible to write to and from the "tags" field in sql drivers
func (t *TagMapData) Scan(src interface{}) error {
	val, ok := src.([]byte)
	if !ok {
		return errors.New("cant scan anything but bytes")
	}
	err := json.Unmarshal(val, &t)
	return err
}

// Value implements the driver.Valuer interface (for writing to db fields)
func (t TagMapData) Value() (driver.Value, error) {
	return json.Marshal(t)
}

// Tags are user-selectable attributes for entities.
type Tags struct {
	TagMap TagMapData
}

// NewTags makes a new Tags instance
func NewTags() Tags {
	return Tags{TagMap: make(map[string]string)}
}

// IsValidTag checks if the name/value combination is valid
func (t *Tags) IsValidTag(name, value string) bool {
	return strings.TrimSpace(name) != ""
}

// SetTag sets a value. If the name already exists it will be overwritten.
// If the value is empty it will be removed
func (t *Tags) SetTag(name, value string) {
	name = strings.TrimSpace(strings.ToLower(name))
	value = strings.TrimSpace(value)

	if value == "" {
		delete(t.TagMap, name)
		return
	}
	t.TagMap[name] = value
}

// GetTag returns the value of the property. If it doesnt't exist it will return
// a blank string
func (t *Tags) GetTag(name string) string {
	return t.TagMap[strings.ToLower(name)]
}

// TagExists returns true if the named value exists
func (t *Tags) TagExists(name string) bool {
	_, ok := t.TagMap[strings.ToLower(name)]
	return ok
}

// TagData returns the tags as a map
func (t *Tags) TagData() map[string]string {
	ret := make(map[string]string)
	for k, v := range t.TagMap {
		ret[k] = v
	}
	return ret
}

// SetTags replaces all tags
func (t *Tags) SetTags(other map[string]string) {
	for k := range t.TagMap {
		delete(t.TagMap, k)
	}
	for k, v := range other {
		t.SetTag(k, v)
	}
}
