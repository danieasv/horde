package fwimage

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
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
)

func testFirmwareStore(t *testing.T, s storage.FirmwareImageStore) {
	id1 := model.FirmwareKey(10001)
	buf1 := make([]byte, 1024)
	rand.Read(buf1)

	h := sha256.New()
	h.Write(buf1)
	cs := hex.EncodeToString(h.Sum(nil))

	if sum, err := s.Create(id1, bytes.NewReader(buf1)); err != nil || sum != cs {
		t.Fatalf("Expected checksum %s but got %s with error %v", cs, sum, err)
	}
	defer s.Delete(id1)

	id2 := model.FirmwareKey(20001)
	buf2 := make([]byte, 128000)
	rand.Read(buf2)
	if _, err := s.Create(id2, bytes.NewReader(buf2)); err != nil {
		t.Fatal(err)
	}
	defer s.Delete(id2)

	if _, err := s.Create(id1, bytes.NewReader(buf2)); err == nil {
		t.Fatal("Expected error when overwriting existing image")
	}

	if _, err := s.Retrieve(model.FirmwareKey(300001)); err == nil {
		t.Fatal("Should not be able to retrieve image that doesn't exist")
	}

	rd1, err := s.Retrieve(id1)
	if err != nil {
		t.Fatal(err)
	}
	defer rd1.Close()

	buf, err := ioutil.ReadAll(rd1)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(buf, buf1) {
		t.Fatal("Buffers are different")
	}

	rd2, err := s.Retrieve(id2)
	if err != nil {
		t.Fatal(err)
	}
	defer rd2.Close()
	buf, err = ioutil.ReadAll(rd2)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(buf, buf2) {
		t.Fatal("Buffers are different")
	}

}
