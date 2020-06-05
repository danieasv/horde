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
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
)

// NewFileSystemStore returns an image store that stores firmware images
// to the local file system. The path must be a writable directory.
//
// Images are stored as <fw id>.image inside the image directory.
func NewFileSystemStore(path string) storage.FirmwareImageStore {
	return &fsStore{imagePath: path}
}

type fsStore struct {
	imagePath string
}

const bufSize = 100 * 1024

func (f *fsStore) Create(fwID model.FirmwareKey, data io.Reader) (string, error) {
	fileName := path.Join(f.imagePath, fmt.Sprintf("%s.image", fwID.String()))
	_, err := os.Stat(fileName)
	if !os.IsNotExist(err) {
		return "", errors.New("image already exists")
	}
	fh, err := os.Create(fileName)
	if err != nil {
		return "", err
	}
	defer fh.Close()

	buf := make([]byte, bufSize)
	h := sha256.New()
	written := 0
	for {
		n, err := data.Read(buf)
		if err != nil && err != io.EOF {
			return "", err
		}
		if n == 0 {
			break
		}
		h.Write(buf[:n])
		if _, err := fh.Write(buf[:n]); err != nil {
			return "", err
		}
		written += n
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func (f *fsStore) Delete(fwID model.FirmwareKey) error {
	fileName := path.Join(f.imagePath, fmt.Sprintf("%s.image", fwID.String()))
	return os.Remove(fileName)
}

func (f *fsStore) Retrieve(fwID model.FirmwareKey) (io.ReadCloser, error) {
	fileName := path.Join(f.imagePath, fmt.Sprintf("%s.image", fwID.String()))
	fh, err := os.OpenFile(fileName, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	return fh, nil
}
