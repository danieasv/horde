package storage
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
import "errors"

var (
	// ErrNotFound is returned when an entity isn't found by the storage layer.
	ErrNotFound = errors.New("entity not found")
	// ErrInternal is returned when there's an (internal) error storing the entity.
	ErrInternal = errors.New("internal storage error")
	// ErrAlreadyExists is returned when the specified entity already exists.
	ErrAlreadyExists = errors.New("entity already exists")
	// ErrAccess is returned when access is denied for an operation
	ErrAccess = errors.New("access denied")
	// ErrNotImplemented is - as the name implies - a placeholder
	ErrNotImplemented = errors.New("feature not implemented")
	// ErrSHAAlreadyExists is returned when a firmware image with that SHA
	// already exists
	ErrSHAAlreadyExists = errors.New("firmware with SHA already exists")
	// ErrReference is returned when there's an reference error deleting or
	// modifying a resource
	ErrReference = errors.New("entity is referenced elsewhere")
)
