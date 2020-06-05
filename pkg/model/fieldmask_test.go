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

func TestFieldMask(t *testing.T) {
	testMasking := func(b FieldMask, imsi, imei, loc, msisdn bool) {
		if !imsi && b.IsSet(IMSIMask) {
			t.Fatalf("IMSI should not be masked (mask = %b)", b)
		}
		if !imei && b.IsSet(IMEIMask) {
			t.Fatalf("IMEI should not be masked (mask = %b)", b)
		}

		if !loc && b.IsSet(LocationMask) {
			t.Fatal("Location should not be masked")
		}
		if !msisdn && b.IsSet(MSISDNMask) {
			t.Fatal("MSISDN should not be masked")
		}
		if imsi && !b.IsSet(IMSIMask) {
			t.Fatal("IMSI should be masked")
		}
		if imei && !b.IsSet(IMEIMask) {
			t.Fatalf("IMEI should be masked (mask = %b)", b)
		}

		if loc && !b.IsSet(LocationMask) {
			t.Fatal("Location should be masked")
		}
		if msisdn && !b.IsSet(MSISDNMask) {
			t.Fatal("MSISDN should be masked")
		}

	}

	editMask := FieldMask(0)

	b := FieldMask(0)

	testMasking(b, false, false, false, false)

	b = 0xFFFF

	testMasking(b, true, true, true, true)

	b.Set(IMSIMask, editMask, false)
	testMasking(b, false, true, true, true)
	b.Set(IMEIMask, editMask, false)
	testMasking(b, false, false, true, true)
	b.Set(MSISDNMask, editMask, false)
	testMasking(b, false, false, true, false)
	b.Set(LocationMask, editMask, false)
	testMasking(b, false, false, false, false)

	b.Set(MSISDNMask, editMask, true)
	testMasking(b, false, false, false, true)
	b.Set(LocationMask, editMask, true)
	testMasking(b, false, false, true, true)
	b.Set(IMEIMask, editMask, true)
	testMasking(b, false, true, true, true)
	b.Set(IMSIMask, editMask, true)
	testMasking(b, true, true, true, true)

	// Change mask config to always mask imsi and imei
	editMask = IMSIMask | IMEIMask
	b = FieldMask(0)
	b.Set(LocationMask, editMask, false)
	testMasking(b, true, true, false, false)
	b.Set(MSISDNMask, editMask, true)
	testMasking(b, true, true, false, true)
	b.Set(MSISDNMask, editMask, false)
	// Should not be updated
	b.Set(IMSIMask, editMask, false)
	b.Set(IMEIMask, editMask, false)
	testMasking(b, true, true, false, false)
}

func TestSummary(t *testing.T) {
	b := MSISDNMask
	n := FieldNames(b)
	if len(n) != 1 || n[0] != "MSISDN" {
		t.Fatalf("Expected 1 element that was MSISDN but it was %v", n)
	}

	b = MSISDNMask | IMSIMask
	n = FieldNames(b)
	if len(n) != 2 || (n[0] != "IMSI" && n[1] != "IMSI") || (n[0] != "MSISDN" && n[1] != "MSISDN") {
		t.Fatalf("Exected 2 elements with IMSI and MSISDN but it was %v", n)
	}

	b = IMEIMask
	n = FieldNames(b)
	if len(n) != 1 || n[0] != "IMEI" {
		t.Fatal("IMEI no work")
	}

	b = LocationMask
	n = FieldNames(b)
	if len(n) != 1 || n[0] != "Location" {
		t.Fatal("Location no work")
	}
}
