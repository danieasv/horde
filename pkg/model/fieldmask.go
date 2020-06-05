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

// FieldMask is the field masking bit used to mask fields
type FieldMask uint

const (
	// IMSIMask is the mask bit for the IMSI field on devices (b0001)
	IMSIMask = FieldMask(1) << iota
	// IMEIMask is the mask bit for the IMEI field on devices (b0010)
	IMEIMask
	// LocationMask is the mask bit for locations on devices (b0100)
	LocationMask
	// MSISDNMask is the mask bit for the MSISDNs on devices (b1000). This isn't
	// currently in use but for political reasons (tm) it is included.
	MSISDNMask
)

// IsSet checks if a field mask bit is set
func (b FieldMask) IsSet(bit FieldMask) bool {
	return b&bit > 0
}

// Set sets a single bit field
func (b *FieldMask) Set(bit FieldMask, editMask FieldMask, set bool) {
	if !set {
		*b = (^bit & *b) | editMask
	} else {
		*b = (bit | *b) | editMask
	}
}

// FieldNames returns the bits that are set as an array of strings. The list
// contains human-readable names
func FieldNames(v FieldMask) []string {
	var ret []string
	if v.IsSet(IMSIMask) {
		ret = append(ret, "IMSI")
	}
	if v.IsSet(IMEIMask) {
		ret = append(ret, "IMEI")
	}
	if v.IsSet(LocationMask) {
		ret = append(ret, "Location")
	}
	if v.IsSet(MSISDNMask) {
		ret = append(ret, "MSISDN")
	}
	return ret
}

// Apply combines two field masks
func (b FieldMask) Apply(systemMask FieldMask) FieldMask {
	return b | systemMask
}
