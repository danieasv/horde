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
	"time"

	"github.com/eesrc/horde/pkg/model"

	"github.com/eesrc/horde/pkg/storage"
)

func testFirmwareStore(e TestEnvironment, s storage.DataStore, t *testing.T) {
	fw11 := model.Firmware{
		ID:           s.NewFirmwareID(),
		Filename:     "fw11",
		Version:      "v11",
		CollectionID: e.C1.ID,
		SHA256:       "aabbccdd",
		Length:       99,
		Created:      time.Now(),
		Tags:         model.NewTags(),
	}
	fw12 := model.Firmware{
		ID:           s.NewFirmwareID(),
		Filename:     "fw12",
		Version:      "v12",
		SHA256:       "aabbccddee",
		Length:       93,
		Created:      time.Now(),
		CollectionID: e.C12.ID,
		Tags:         model.NewTags(),
	}
	fw21 := model.Firmware{
		ID:           s.NewFirmwareID(),
		Filename:     "fw21",
		Version:      "v21",
		SHA256:       "aabbccddff",
		Length:       92,
		Created:      time.Now(),
		CollectionID: e.C2.ID,
		Tags:         model.NewTags(),
	}
	fw22 := model.Firmware{
		ID:           s.NewFirmwareID(),
		Version:      "v22",
		Filename:     "fw22",
		SHA256:       "aabbccddeeff",
		Length:       94,
		Created:      time.Now(),
		CollectionID: e.C21.ID,
		Tags:         model.NewTags(),
	}

	list, err := s.ListFirmware(e.U3.ID, e.C3.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) > 0 {
		t.Fatal("Did not expect any firmware images in the list")
	}

	if err := s.CreateFirmware(e.U1.ID, fw11); err != nil {
		t.Fatal(err)
	}

	if err := s.CreateFirmware(e.U1.ID, fw12); err != nil {
		t.Fatal(err)
	}

	if err := s.CreateFirmware(e.U1.ID, fw21); err == storage.ErrAccess {
		t.Fatal("Should not be able to create firmware for teams where I'm not the owner")
	}

	fwDup := fw11
	fwDup.SHA256 = "ccddee"
	if err := s.CreateFirmware(e.U1.ID, fwDup); err != storage.ErrAlreadyExists {
		t.Fatal("Epected already exists error but got ", err)
	}

	if err := s.CreateFirmware(e.U2.ID, fw21); err != nil {
		t.Fatal(err)
	}

	if err := s.CreateFirmware(e.U2.ID, fw22); err != nil {
		t.Fatal(err)
	}
	list, err = s.ListFirmware(e.U1.ID, e.C1.ID)
	if err != nil {
		t.Fatal(err)
	}
	// Should have 11
	if len(list) != 1 {
		t.Fatal("Expected 1 images but got ", len(list))
	}
	if list[0].ID != fw11.ID {
		t.Fatal("Expected fw11 in list")
	}

	list, err = s.ListFirmware(e.U1.ID, e.C12.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatal("Expected 1 images but got ", len(list))
	}
	if list[0].ID != fw12.ID {
		t.Fatal("Expected fw12 in list")
	}

	fw, err := s.RetrieveFirmware(e.U1.ID, e.C1.ID, fw11.ID)
	if err != nil {
		t.Fatal(err)
	}
	if fw.Created.UnixNano() != fw11.Created.UnixNano() {
		t.Fatal("Time is different")
	}
	// Time zone information are slightly different here but that's OK since the
	// actual time stamp is the same
	fw.Created = fw11.Created
	if !reflect.DeepEqual(fw, fw11) {
		t.Fatalf("Retrieved image and image are different: %+v != %+v", fw, fw11)
	}

	if _, err := s.RetrieveFirmware(e.U1.ID, e.C21.ID, fw21.ID); err != storage.ErrNotFound {
		t.Fatal("Did not get not found error but got ", err)
	}

	// Update firmware image fw21
	fw21.SetTag("name", "the new")
	fw21.CollectionID = e.C2.ID
	if err := s.UpdateFirmware(e.U2.ID, e.C21.ID, fw21); err != nil {
		t.Fatal(err)
	}
	fwB, err := s.RetrieveFirmware(e.U2.ID, e.C2.ID, fw21.ID)
	if err != nil {
		t.Fatal(err)
	}
	if fwB.GetTag("name") != "the new" {
		t.Fatalf("Tag isn't set correctly tags = %+v", fwB.TagData())
	}
	if fwB.CollectionID != e.C2.ID {
		t.Fatal("Team ID isn't set correctly")
	}
	fw12.CollectionID = e.C21.ID
	if err := s.UpdateFirmware(e.U1.ID, e.C12.ID, fw12); err != storage.ErrAccess {
		t.Fatal("Expected ErrAccess when attempting to transfer firmware to non-admin team but got ", err)
	}

	testCreateDuplicateImage(t, e, s)
	testCurrentAndNewFirmware(t, e, s)
	testFirmwareConfig(t, e, s)

	testFirmwareInUse(t, e, s)

	testTagSetAndGet(t, e, fw12.ID.String(), true, s.UpdateFirmwareTags, s.RetrieveFirmwareTags)

	if err := s.DeleteFirmware(e.U2.ID, e.C12.ID, fw12.ID); err != storage.ErrAccess {
		t.Fatal("Did not get access error but got ", err)
	}
	if err := s.DeleteFirmware(e.U1.ID, e.C12.ID, fw12.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteFirmware(e.U1.ID, e.C12.ID, fw12.ID); err != storage.ErrNotFound {
		t.Fatal("Expected not found but got ", err)
	}
	s.DeleteFirmware(e.U1.ID, e.C1.ID, fw11.ID)
	s.DeleteFirmware(e.U2.ID, e.C21.ID, fw21.ID)
	s.DeleteFirmware(e.U2.ID, e.C2.ID, fw22.ID)
}

// Ensure firmware images are unique in collection
func testCreateDuplicateImage(t *testing.T, e TestEnvironment, s storage.DataStore) {
	fw := model.Firmware{
		ID:           s.NewFirmwareID(),
		Version:      "9.0",
		Filename:     "fileX",
		Length:       100,
		SHA256:       "beefbabebeefbabe",
		Created:      time.Now(),
		CollectionID: e.C1.ID,
	}
	if err := s.CreateFirmware(e.U1.ID, fw); err != nil {
		t.Fatal(err)
	}

	fwCopy := fw
	fwCopy.ID = s.NewFirmwareID()
	fwCopy.Version = "3.0"
	fwCopy.Filename = "newFileButSame"
	fwCopy.Created = time.Now()

	if err := s.CreateFirmware(e.U1.ID, fwCopy); err != storage.ErrSHAAlreadyExists {
		t.Fatal("Expected already exists but got ", err)
	}
	defer s.DeleteFirmware(e.U1.ID, fw.CollectionID, fw.ID)
}

// Test the RetrieveCurrentAndTarget methods
func testCurrentAndNewFirmware(t *testing.T, e TestEnvironment, s storage.DataStore) {
	fwX := model.Firmware{
		ID:           s.NewFirmwareID(),
		Version:      "1.0",
		Filename:     "fileX",
		Length:       100,
		SHA256:       "ddeeddee",
		Created:      time.Now(),
		CollectionID: e.C1.ID,
	}
	if err := s.CreateFirmware(e.U1.ID, fwX); err != nil {
		t.Fatal(err)
	}

	fwY := model.Firmware{
		ID:           s.NewFirmwareID(),
		Version:      "1.1",
		Filename:     "fileY",
		Length:       200,
		SHA256:       "aabbccddeeaa",
		Created:      time.Now(),
		CollectionID: e.C1.ID,
	}
	if err := s.CreateFirmware(e.U1.ID, fwY); err != nil {
		t.Fatal(err)
	}

	fwA, fwB, err := s.RetrieveCurrentAndTargetFirmware(e.C1.ID, fwX.ID, fwY.ID)
	if err != nil {
		t.Fatal(err)
	}
	if fwA.ID != fwX.ID {
		t.Fatalf("Expected X to be returned as first element but got %+v", fwA)
	}
	if fwB.ID != fwY.ID {
		t.Fatalf("Expected Y to be returned as second element but got %+v", fwB)
	}

	_, _, err = s.RetrieveCurrentAndTargetFirmware(e.C2.ID, s.NewFirmwareID(), fwY.ID)
	if err != storage.ErrNotFound {
		t.Fatalf("Expected NotFound but got %d", err)
	}

	_, _, err = s.RetrieveCurrentAndTargetFirmware(e.C1.ID, s.NewFirmwareID(), s.NewFirmwareID())
	if err != storage.ErrNotFound {
		t.Fatalf("Expected NotFound but got %d", err)
	}

	if err := s.DeleteFirmware(e.U1.ID, e.C1.ID, fwX.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteFirmware(e.U1.ID, e.C1.ID, fwY.ID); err != nil {
		t.Fatal(err)
	}
}

func testFirmwareConfig(t *testing.T, e TestEnvironment, s storage.DataStore) {
	fwA := model.Firmware{
		ID:           s.NewFirmwareID(),
		Version:      "1.0",
		Filename:     "fileX",
		Length:       100,
		SHA256:       "ccbbddeeaadd",
		Created:      time.Now(),
		CollectionID: e.C1.ID,
	}
	if err := s.CreateFirmware(e.U1.ID, fwA); err != nil {
		t.Fatal(err)
	}

	fwB := model.Firmware{
		ID:           s.NewFirmwareID(),
		Version:      "1.1",
		Filename:     "fileY",
		Length:       200,
		SHA256:       "aabbccddeeaa",
		Created:      time.Now(),
		CollectionID: e.C1.ID,
	}
	if err := s.CreateFirmware(e.U1.ID, fwB); err != nil {
		t.Fatal(err)
	}
	fwC := model.Firmware{
		ID:           s.NewFirmwareID(),
		Version:      "1.2",
		Filename:     "fileZ",
		Length:       300,
		SHA256:       "aaccddeeaabb",
		Created:      time.Now(),
		CollectionID: e.C1.ID,
	}
	if err := s.CreateFirmware(e.U1.ID, fwC); err != nil {
		t.Fatal(err)
	}
	fwD := model.Firmware{
		ID:           s.NewFirmwareID(),
		Version:      "1.3",
		Filename:     "fileY",
		Length:       200,
		SHA256:       "baefbaefbaef",
		Created:      time.Now(),
		CollectionID: e.C1.ID,
	}
	if err := s.CreateFirmware(e.U1.ID, fwD); err != nil {
		t.Fatal(err)
	}

	d := model.NewDevice()
	d.ID = s.NewDeviceID()
	d.IMSI = 39399393992
	d.IMEI = d.IMSI
	d.Firmware.CurrentFirmwareID = fwC.ID
	d.Firmware.TargetFirmwareID = fwD.ID
	d.CollectionID = e.C1.ID

	if err := s.CreateDevice(e.U1.ID, d); err != nil {
		t.Fatal(err)
	}

	e.C1.Firmware.CurrentFirmwareID = fwA.ID
	e.C1.Firmware.TargetFirmwareID = fwB.ID
	e.C1.Firmware.Management = model.CollectionManagement
	if err := s.UpdateCollection(e.U1.ID, e.C1); err != nil {
		t.Fatal(err)
	}

	cfg, err := s.RetrieveFirmwareConfig(e.C1.ID, d.ID)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CollectionCurrentVersion != e.C1.Firmware.CurrentFirmwareID ||
		cfg.CollectionTargetVersion != e.C1.Firmware.TargetFirmwareID {
		t.Fatal("Collection fw version is incorrect")
	}
	if cfg.DeviceCurrentVersion != d.Firmware.CurrentFirmwareID ||
		cfg.DeviceTargetVersion != d.Firmware.TargetFirmwareID {
		t.Fatal("Device fw version is incorrect")
	}
	if cfg.Management != e.C1.Firmware.Management {
		t.Fatal("Management setting is incorrect")
	}
}

func testFirmwareInUse(t *testing.T, e TestEnvironment, s storage.DataStore) {
	fw1 := model.Firmware{
		ID:           s.NewFirmwareID(),
		Version:      "2.0",
		Filename:     "f1",
		SHA256:       "abcdefabcdef11223344",
		Created:      time.Now(),
		CollectionID: e.C1.ID,
		Tags:         model.NewTags(),
	}
	fw2 := model.Firmware{
		ID:           s.NewFirmwareID(),
		Version:      "3.0",
		Filename:     "f2",
		SHA256:       "abcdefabcdef1122334455",
		Created:      time.Now(),
		CollectionID: e.C1.ID,
		Tags:         model.NewTags(),
	}

	if err := s.CreateFirmware(e.U1.ID, fw1); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateFirmware(e.U1.ID, fw2); err != nil {
		t.Fatal(err)
	}

	defer s.DeleteFirmware(e.U1.ID, e.C1.ID, fw1.ID)
	defer s.DeleteFirmware(e.U1.ID, e.C1.ID, fw2.ID)

	// Firmware images should not be in use
	use1, err := s.RetrieveFirmwareVersionsInUse(e.U1.ID, e.C1.ID, fw1.ID)
	if err != nil {
		t.Fatal(err)
	}
	if use1.FirmwareID != fw1.ID {
		t.Fatal("")
	}
	if len(use1.Current) != 0 {
		t.Fatal("Expected 0 images in use", use1)
	}
	if len(use1.Targeted) != 0 {
		t.Fatal("Expected 0 images targeted", use1)
	}

	// Create two devices with firmware versions set
	d1 := model.Device{
		ID:           s.NewDeviceID(),
		IMSI:         int64(s.NewDeviceID()),
		IMEI:         int64(s.NewDeviceID()),
		Tags:         model.NewTags(),
		CollectionID: e.C1.ID,
		Firmware: model.DeviceFirmwareMetadata{
			CurrentFirmwareID: fw1.ID,
			TargetFirmwareID:  fw2.ID,
		},
	}
	d2 := model.Device{
		ID:           s.NewDeviceID(),
		IMSI:         int64(s.NewDeviceID()),
		IMEI:         int64(s.NewDeviceID()),
		Tags:         model.NewTags(),
		CollectionID: e.C1.ID,
		Firmware: model.DeviceFirmwareMetadata{
			CurrentFirmwareID: fw1.ID,
		},
	}
	if err := s.CreateDevice(e.U1.ID, d1); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateDevice(e.U1.ID, d2); err != nil {
		t.Fatal(err)
	}

	use1, err = s.RetrieveFirmwareVersionsInUse(e.U1.ID, e.C1.ID, fw1.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(use1.Current) != 2 {
		t.Fatal("Expected 2 current devices", use1)
	}
	if len(use1.Targeted) != 0 {
		t.Fatal("Expected 0 targeted devices", use1)
	}

	use2, err := s.RetrieveFirmwareVersionsInUse(e.U1.ID, e.C1.ID, fw2.ID)
	if err != nil {
		t.Fatal(err)
	}
	if use2.FirmwareID != fw2.ID {
		t.Fatal(use2)
	}
	if len(use2.Current) != 0 {
		t.Fatal("Expected 0 current devices", use2)
	}
	if len(use2.Targeted) != 1 {
		t.Fatal("Expected 1 targeted device", use2)
	}

	use3, err := s.RetrieveFirmwareVersionsInUse(e.U2.ID, e.C1.ID, fw2.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(use3.Current) != 0 {
		t.Fatal(err)
	}

	s.DeleteDevice(e.U1.ID, e.C1.ID, d1.ID)
	s.DeleteDevice(e.U1.ID, e.C1.ID, d2.ID)
}
