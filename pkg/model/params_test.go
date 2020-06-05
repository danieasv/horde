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
import "testing"

func TestFieldMaskParameters(t *testing.T) {
	p := FieldMaskParameters{
		Forced:  "msisdn",
		Default: "location,msisdn",
	}

	if p.Valid() != nil {
		t.Fatal("Should be valid")
	}

	p.Forced = "msisdn,location"
	p.Default = "location"
	if p.Valid() == nil {
		t.Fatal("Shouldn't be valid")
	}

	p.Forced = "msisdn"
	p.Default = "imsi"
	if p.Valid() == nil {
		t.Fatal("Shouldn't be valid")
	}

	p.Forced = "msisdn,imsi,imei"
	p.Default = "location"
	if p.Valid() == nil {
		t.Fatal("Shouldn't be valid")
	}

	p.Default = "msisdn,imsi,imei,location"
	if p.Valid() != nil {
		t.Fatal("Should be valid")
	}

	p.Default = ""
	p.Forced = ""
	if err := p.Valid(); err != nil {
		t.Fatal("Should be valid:", err)
	}

	p.Default = "something,imsi"
	p.Forced = "imsi"
	if p.Valid() == nil {
		t.Fatal("Shouldn't be valid")
	}
	p.Default = ",,,,,,,,,,,"
	p.Forced = ",,"
	if p.Valid() == nil {
		t.Fatal("Shouldn't be valid")
	}
}
