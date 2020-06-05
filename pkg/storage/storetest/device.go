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
	"fmt"
	"testing"
	"time"

	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
	"github.com/stretchr/testify/require"
)

// TestDeviceStore runs a series on tests on a DeviceStore implementation
func testDeviceStore(env TestEnvironment, s storage.DataStore, t *testing.T) {

	d1 := make([]model.Device, 0)
	d2 := make([]model.Device, 0)
	d12 := make([]model.Device, 0)
	d21 := make([]model.Device, 0)
	const numDevices = 10

	// Create devices for U1 and U2 via collections C1, C2, C12, C23
	for i := 0; i < numDevices; i++ {
		d := model.NewDevice()
		d.ID = s.NewDeviceID()
		d.IMEI = int64(d.ID)
		d.IMSI = int64(d.ID)
		d.Firmware = model.DeviceFirmwareMetadata{
			ModelNumber:  "100",
			SerialNumber: "100",
			Manufacturer: "EE",
		}
		d.Network = model.DeviceNetworkMetadata{
			AllocatedIP: "127.0.0.1",
			AllocatedAt: time.Now(),
			CellID:      1000,
			ApnID:       20,
			NasID:       10,
		}
		d.SetTag("name", fmt.Sprintf("Device %d for user 1", i))
		d.CollectionID = env.C1.ID
		if err := s.CreateDevice(env.U1.ID, d); err != nil {
			t.Fatalf("1: Unable to create device #%d: %v", i, err)
		}
		d1 = append(d1, d)

		d = model.NewDevice()
		d.ID = s.NewDeviceID()
		d.IMEI = int64(d.ID)
		d.IMSI = int64(d.ID)
		d.SetTag("name", fmt.Sprintf("Device %d for user 2", i))
		d.CollectionID = env.C2.ID
		if err := s.CreateDevice(env.U2.ID, d); err != nil {
			t.Fatalf("2: Unable to create device #%d: %v", i, err)
		}
		d2 = append(d2, d)

		d = model.NewDevice()
		d.ID = s.NewDeviceID()
		d.IMEI = int64(d.ID)
		d.IMSI = int64(d.ID)
		d.SetTag("name", fmt.Sprintf("Device %d for user 1&2", i))
		d.CollectionID = env.C12.ID
		if err := s.CreateDevice(env.U1.ID, d); err != nil {
			t.Fatalf("1&2: Unable to create device #%d: %v", i, err)
		}
		d12 = append(d12, d)

		d = model.NewDevice()
		d.ID = s.NewDeviceID()
		d.IMEI = int64(d.ID)
		d.IMSI = int64(d.ID)
		d.SetTag("name", fmt.Sprintf("Device %d for user 2&1", i))
		d.CollectionID = env.C21.ID
		if err := s.CreateDevice(env.U2.ID, d); err != nil {
			t.Fatalf("2&1: Unable to create device #%d: %v", i, err)
		}
		d21 = append(d21, d)
	}
	d := model.NewDevice()
	d.ID = s.NewDeviceID()
	d.CollectionID = env.C21.ID
	if err := s.CreateDevice(env.U1.ID, d); err != storage.ErrAccess {
		t.Fatal("Should not be allowed to create a device for a non-admin collection. err = ", err)
	}
	d.CollectionID = env.C12.ID
	if err := s.CreateDevice(env.U2.ID, d); err != storage.ErrAccess {
		t.Fatal("Should not be allowed to create a device for a non-admin collection. err = ", err)
	}
	d.CollectionID = env.C3.ID
	if err := s.CreateDevice(env.U1.ID, d); err != storage.ErrNotFound {
		t.Fatal("Should not be allowed to create a device for a non-admin collection. err = ", err)
	}

	if err := s.CreateDevice(env.U1.ID, d1[4]); err != storage.ErrAlreadyExists {
		t.Fatal("Should not be allowed to create a device with existing ID. err = ", err)
	}
	d.ID = s.NewDeviceID()
	d.CollectionID = env.C1.ID
	d.IMEI = d2[0].IMEI
	if err := s.CreateDevice(env.U1.ID, d); err != storage.ErrAlreadyExists {
		t.Fatal("Should not be allowed to create a device with existing IMEI. err = ", err)
	}
	d.IMEI = int64(d.ID)
	d.IMSI = d2[0].IMSI
	if err := s.CreateDevice(env.U1.ID, d); err != storage.ErrAlreadyExists {
		t.Fatal("Should not be allowed to create a device with existing IMSI. err = ", err)
	}

	// check returned list of devices
	hasElements := func(arr []model.Device, ids ...model.DeviceKey) bool {
		if arr == nil || len(arr) < len(ids) {
			return false
		}
		found := 0
		for _, v := range ids {
			for i := range arr {
				if arr[i].ID == v {
					found++
					break
				}
			}
		}
		return found == len(ids)
	}

	testList := func(u model.UserKey, c model.CollectionKey, l []model.Device) {
		l1, err := s.ListDevices(u, c)
		if err != nil {
			t.Fatal("Unable to retrieve list of devices: ", err)
		}

		keys := make([]model.DeviceKey, len(l))
		for i := range l {
			keys[i] = l[i].ID
		}
		if !hasElements(l1, keys...) {
			t.Fatal("Missing devices from list")
		}
	}

	testList(env.U1.ID, env.C1.ID, d1)
	testList(env.U1.ID, env.C12.ID, d12)
	testList(env.U2.ID, env.C12.ID, d12)
	testList(env.U2.ID, env.C2.ID, d2)
	testList(env.U1.ID, env.C21.ID, d21)
	testList(env.U2.ID, env.C21.ID, d21)

	if _, err := s.ListDevices(env.U1.ID, env.C3.ID); err != storage.ErrNotFound {
		t.Fatal("Should get not found error but got ", err)
	}

	// Retrieve devices in order
	for i := range d1 {
		d, err := s.RetrieveDevice(env.U1.ID, d1[i].CollectionID, d1[i].ID)
		if err != nil {
			t.Fatalf("Unable to retrieve device %s: %v", d.ID, err)
		}
	}
	for i := range d2 {
		d, err := s.RetrieveDevice(env.U2.ID, d2[i].CollectionID, d2[i].ID)
		if err != nil {
			t.Fatalf("Unable to retrieve device %s: %v", d.ID, err)
		}
	}
	if _, err := s.RetrieveDevice(env.U1.ID, d2[0].CollectionID, d2[0].ID); err != storage.ErrNotFound {
		t.Fatal("Should not be able to access devices the user doesn't own. err = ", err)
	}
	if _, err := s.RetrieveDevice(env.U1.ID, d1[0].CollectionID, s.NewDeviceID()); err != storage.ErrNotFound {
		t.Fatal("Should not be able to retrieve devices that doesn't work. err = ", err)
	}

	testTagSetAndGet(t, env, d12[0].ID.String(), true, s.UpdateDeviceTags, s.RetrieveDeviceTags)

	// Attempt an update on device
	d = d1[0]
	d.IMEI = int64(s.NewDeviceID())
	d.IMSI = int64(s.NewDeviceID())
	if err := s.UpdateDevice(env.U1.ID, env.C1.ID, d); err != nil {
		t.Fatal("Should be able to update a device with the correct owner, err:", err)
	}
	d.IMEI = int64(s.NewDeviceID())
	d.IMSI = int64(s.NewDeviceID())
	if err := s.UpdateDevice(env.U2.ID, env.C1.ID, d); err != storage.ErrNotFound {
		t.Fatal("Should not be able to update a device with the incorrect owner. err = ", err)
	}
	d.CollectionID = env.C2.ID
	if err := s.UpdateDevice(env.U2.ID, env.C2.ID, d); err != storage.ErrNotFound {
		t.Fatal("Should not be able to update a device with the incorrect owner. err = ", err)
	}
	d.CollectionID = env.C2.ID
	if err := s.UpdateDevice(env.U1.ID, env.C1.ID, d); err != storage.ErrNotFound {
		t.Fatal("Should not be able to update device with new collection that I don't own. err = ", err)
	}
	d.CollectionID = env.C21.ID
	if err := s.UpdateDevice(env.U1.ID, env.C1.ID, d); err != storage.ErrAccess {
		t.Fatal("Should not be able to update device with new collection where I'm not admin. err = ", err)
	}
	d.ID = d1[0].ID
	d.CollectionID = env.C1.ID
	d.IMEI = d1[1].IMEI
	if err := s.UpdateDevice(env.U1.ID, env.C1.ID, d); err != storage.ErrAlreadyExists {
		t.Fatal("Should not be able to update with existing IMEI. err = ", err)
	}
	d.IMEI = d1[0].IMEI
	d.IMSI = d1[1].IMSI
	if err := s.UpdateDevice(env.U1.ID, env.C1.ID, d); err != storage.ErrAlreadyExists {
		t.Fatal("Should not be able to update with existing IMSI")
	}

	d.IMEI = int64(s.NewDeviceID())
	d.IMSI = int64(s.NewDeviceID())
	d.ID = s.NewDeviceID()
	if err := s.UpdateDevice(env.U1.ID, env.C1.ID, d); err != storage.ErrNotFound {
		t.Fatal("Should not be able to update device that doesn't exist")
	}

	// Change category ID for one device. From C1 -> C12 is OK. C1 -> C21 should not work
	var err error
	const index = 5
	if d, err = s.RetrieveDeviceByIMSI(d1[index].IMSI); err != nil {
		t.Fatalf("Should be able to retrieve device with IMSI %d: %v (id= %d)", d1[index].IMSI, err, d1[index].ID)
	}
	// Change IMSI is OK
	oldIMSI := d.IMSI
	d.IMSI = 99999999999
	if err := s.UpdateDevice(env.U1.ID, env.C1.ID, d); err != nil {
		t.Fatalf("Should be allowed to change device to non-existing IMSI: %v", err)
	}
	var dnew model.Device
	if dnew, err = s.RetrieveDeviceByIMSI(d.IMSI); err != nil {
		t.Fatalf("Should be able to retrieve device with new IMSI: %v", err)
	}
	if dnew.ID != d.ID {
		t.Fatalf("Device isn't the same")
	}
	// The *old* IMSI shouldn't return anything
	if dev, err := s.RetrieveDeviceByIMSI(oldIMSI); err != storage.ErrNotFound {
		t.Fatalf("Should not be able to retrieve device with old IMSI but got device: %+v, (err = %v)", dev, err)
	}
	// Category change is ok
	d.CollectionID = env.C12.ID
	if err := s.UpdateDevice(env.U1.ID, env.C1.ID, d); err != nil {
		t.Fatalf("Unable to change category from C1 -> C12 for device: %v", err)
	}
	d1[0] = d
	d.CollectionID = env.C21.ID
	if err := s.UpdateDevice(env.U1.ID, env.C12.ID, d); err == nil {
		t.Fatalf("Should not be able to move device to collection I don't administer")
	}
	if dev, err := s.RetrieveDevice(env.U1.ID, env.C1.ID, d.ID); err != storage.ErrNotFound {
		t.Fatalf("Should not be able to look up via old collection id (dev=%+v, err=%v)", dev, err)
	}
	// Do a direct update of tags
	dnew.SetTag("foof", "Some new tag")
	if err := s.UpdateDeviceMetadata(dnew); err != nil {
		t.Fatalf("Should be able to update tags on device (system version): %v", err)
	}
	dnew.ID = 9999
	if err := s.UpdateDeviceMetadata(dnew); err == nil {
		t.Fatal("Should not able to update tags on unknown device (system version) but no error returned")
	}

	testMetadataUpdate(t, s, env, d1[9])
	testStateUpdate(t, s, env, d1[8])

	// ...and delete
	if err := s.DeleteDevice(env.U1.ID, d2[0].CollectionID, d2[0].ID); err != storage.ErrNotFound {
		t.Fatal("Should not be able to delete device that user doesn't own. err = ", err)
	}
	if err := s.DeleteDevice(env.U1.ID, d1[0].CollectionID, d2[0].ID); err != storage.ErrNotFound {
		t.Fatal("Should not be able to delete device that user doesn't own. err = ", err)
	}
	if err := s.DeleteDevice(env.U1.ID, d2[0].CollectionID, d1[0].ID); err != storage.ErrNotFound {
		t.Fatal("Should not be able to delete device with incorrect collection id. err = ", err)
	}
	if err := s.DeleteDevice(env.U1.ID, d21[0].CollectionID, d21[0].ID); err != storage.ErrAccess {
		t.Fatal("Should not be able to delete device that user isn't admin for. err = ", err)
	}
	if err := s.DeleteDevice(env.U1.ID, d1[0].CollectionID, d1[0].ID); err != nil {
		t.Fatal("Should be able to delete device: ", err)
	}
}

func testMetadataUpdate(t *testing.T, s storage.DataStore, env TestEnvironment, d model.Device) {
	fw1 := model.Firmware{
		ID:           s.NewFirmwareID(),
		Version:      "1.0",
		Filename:     "f1",
		SHA256:       "abcdefabcdef1122334422",
		Created:      time.Now(),
		CollectionID: env.C1.ID,
		Tags:         model.NewTags(),
	}
	fw2 := model.Firmware{
		ID:           s.NewFirmwareID(),
		Version:      "2.0",
		Filename:     "f2",
		SHA256:       "abcdefabcdef112233445511",
		Created:      time.Now(),
		CollectionID: env.C1.ID,
		Tags:         model.NewTags(),
	}

	if err := s.CreateFirmware(env.U1.ID, fw1); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateFirmware(env.U1.ID, fw2); err != nil {
		t.Fatal(err)
	}

	d.Network.AllocatedIP = "127.0.0.1"
	d.Network.AllocatedAt = time.Now()
	d.Network.CellID = 10000
	d.Network.ApnID = 10
	d.Network.NasID = 5

	d.Firmware.SerialNumber = "2000"
	d.Firmware.ModelNumber = "2000"
	d.Firmware.Manufacturer = "EE"
	d.Firmware.FirmwareVersion = "1.0.0"
	d.Firmware.CurrentFirmwareID = fw1.ID
	d.Firmware.TargetFirmwareID = fw2.ID

	if err := s.UpdateDeviceMetadata(d); err != nil {
		t.Fatal(err)
	}

	d.Network.AllocatedIP = ""
	d.Network.AllocatedAt = time.Unix(0, 0)
	d.Network.CellID = 0
	d.Network.ApnID = 0
	d.Network.NasID = 0

	d.Firmware.SerialNumber = ""
	d.Firmware.ModelNumber = ""
	d.Firmware.Manufacturer = ""
	d.Firmware.FirmwareVersion = ""
	d.Firmware.CurrentFirmwareID = 0
	d.Firmware.TargetFirmwareID = 0

	if err := s.UpdateDeviceMetadata(d); err != nil {
		t.Fatal(err)
	}

	s.DeleteFirmware(env.U1.ID, env.C1.ID, fw1.ID)
	s.DeleteFirmware(env.U1.ID, env.C1.ID, fw2.ID)

}

func testStateUpdate(t *testing.T, s storage.DataStore, env TestEnvironment, d model.Device) {
	assert := require.New(t)

	assert.NoError(s.UpdateFirmwareStateForDevice(d.IMSI, model.Current, "device is up to date"))

	newD, err := s.RetrieveDevice(env.U1.ID, env.C1.ID, d.ID)
	assert.NoError(err)
	assert.Equal(model.Current, newD.Firmware.State)
	assert.Equal("device is up to date", newD.Firmware.StateMessage)

	assert.Error(storage.ErrNotFound, s.UpdateFirmwareStateForDevice(-1, model.Initializing, ""))
}
