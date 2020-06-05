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
	"errors"
	"fmt"
	"strings"
)

const (
	imsiField     = "imsi"
	imeiField     = "imei"
	locationField = "location"
	msisdnField   = "msisdn"
)

func stringToFieldMask(mask string) (FieldMask, error) {
	fields := strings.Split(mask, ",")
	ret := FieldMask(0)
	if len(fields) == 0 || len(fields) == 1 && fields[0] == "" {
		return ret, nil
	}
	for _, v := range fields {
		switch strings.TrimSpace(strings.ToLower(v)) {
		case imsiField:
			ret.Set(IMSIMask, 0, true)
		case imeiField:
			ret.Set(IMEIMask, 0, true)
		case locationField:
			ret.Set(LocationMask, 0, true)
		case msisdnField:
			ret.Set(MSISDNMask, 0, true)
		default:
			return ret, fmt.Errorf("unknown field: %s", v)
		}
	}
	return ret, nil
}

// FieldMaskParameters contains the system configuration for the field masks;
// system defaults and edit mask. The edit mask sets the allowed edits, in
// practice making the field mask read only for certain fields.
//
// The EditMask field has bits set for all fields that must be masked, ie if
// set to 0xFFFF all fields will be masked and the API clients are unable to
// modfiy the mask
type FieldMaskParameters struct {
	Forced  string `param:"desc=Read-only field mask for device fields"`
	Default string `param:"desc=Default field mask for device fields;default=location"`
}

// Valid validates the forced and default field masks. The default field
// mask must be equal or more restrictive than the forced field mask.
func (f *FieldMaskParameters) Valid() error {
	if _, err := stringToFieldMask(f.Forced); err != nil {
		return err
	}
	if _, err := stringToFieldMask(f.Default); err != nil {
		return err
	}
	if int(f.DefaultFields())-int(f.ForcedFields()) < 0 {
		return errors.New("forced field mask must be less restrictive than the default field mask")
	}
	return nil
}

// ForcedFields returns the field mask for forced fields
func (f *FieldMaskParameters) ForcedFields() FieldMask {
	ret, _ := stringToFieldMask(f.Forced)
	return ret
}

// DefaultFields returns the default field mask
func (f *FieldMaskParameters) DefaultFields() FieldMask {
	ret, _ := stringToFieldMask(f.Default)
	return ret
}
