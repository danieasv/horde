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
	"errors"
	"io"

	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
)

type grpcStore struct {
}

// NewGRPCStore creates a new firmware image store that uses a gRPC service
// to store images
func NewGRPCStore() storage.FirmwareImageStore {
	return &grpcStore{}
}

func (g *grpcStore) Create(model.FirmwareKey, io.Reader) (string, error) {
	sha := ""
	err := errors.New("not implemented")

	return sha, err
}

func (g *grpcStore) Retrieve(model.FirmwareKey) (io.ReadCloser, error) {
	return nil, errors.New("not implemented")
}

func (g *grpcStore) Delete(model.FirmwareKey) error {
	return errors.New("not implemented")
}
