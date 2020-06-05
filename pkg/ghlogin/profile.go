package ghlogin

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
)

// Profile contains the GitHub profile
type Profile struct {
	AvatarURL string `json:"avatarUrl"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Login     string `json:"loginName"`
}

// Scan implements the sql.Scanner interface (to read from db fields). This makes
// it possible to write to and from the "tags" field in sql drivers
func (p *Profile) Scan(src interface{}) error {
	val, ok := src.([]byte)
	if !ok {
		return errors.New("cant scan anything but bytes")
	}
	return json.Unmarshal(val, p)
}

// Value implements the driver.Valuer interface (for writing to db fields)
func (p *Profile) Value() (driver.Value, error) {
	return json.Marshal(p)
}

// protfileFromMap reads the profile from a the dicitionary returned by the
// GitHub API.
func profileFromMap(userProfile map[string]interface{}) Profile {
	mapString := func(dict map[string]interface{}, name string) string {
		ret, ok := dict[name]
		if !ok {
			return ""
		}
		val, ok := ret.(string)
		if !ok {
			return ""
		}
		return val
	}

	ret := Profile{}
	ret.AvatarURL = mapString(userProfile, "avatar_url")
	ret.Email = mapString(userProfile, "email")
	ret.Login = mapString(userProfile, "login")
	ret.Name = mapString(userProfile, "name")
	return ret
}
