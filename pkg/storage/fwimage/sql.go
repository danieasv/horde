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
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"io"
	"io/ioutil"
	"strings"

	"github.com/ExploratoryEngineering/logging"

	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
	"github.com/eesrc/horde/pkg/storage/sqlstore"
)

// MaxImageSize is the maximum image size allowed. Size is TBD.
const MaxImageSize = 2 * 1024 * 1024

type sqlStore struct {
	db           *sql.DB
	createStmt   *sql.Stmt
	retrieveStmt *sql.Stmt
	deleteStmt   *sql.Stmt
}

// NewSQLStore creates a new firmware image store that stores images in a database
func NewSQLStore(params sqlstore.Parameters) (storage.FirmwareImageStore, error) {
	db, err := sql.Open(params.Type, params.ConnectionString)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	ret := &sqlStore{db: db}
	if err := ret.createSchema(params); err != nil {
		logging.Error("Unable to create schema for firmware image store: %v", err)
		return nil, err
	}

	if err := ret.prepare(); err != nil {
		logging.Error("Unable to prepare statements for firmware image store: %v", err)
		return nil, err
	}
	return ret, nil
}

func (s *sqlStore) createSchema(params sqlstore.Parameters) error {
	create := `
	CREATE TABLE IF NOT EXISTS firmware_image (
		image_id   BIGINT      NOT NULL,
		image_data BYTEA       NOT NULL,

		CONSTRAINT firmware_image_pk PRIMARY KEY (image_id)
	);
	`
	schema := sqlstore.NewSchema(params.Type, create)
	for _, v := range schema.Statements() {
		if _, err := s.db.Exec(v); err != nil {
			return err
		}
	}
	return nil
}

func (s *sqlStore) prepare() error {
	var err error
	if s.retrieveStmt, err = s.db.Prepare(`
	SELECT image_data FROM firmware_image WHERE image_id = $1
`); err != nil {
		return err
	}
	if s.createStmt, err = s.db.Prepare(`
	INSERT INTO firmware_image (image_id, image_data)
	VALUES ($1, $2)
`); err != nil {
		return err
	}
	if s.deleteStmt, err = s.db.Prepare(`
	DELETE FROM firmware_image WHERE image_id = $1
`); err != nil {
		return err
	}
	return nil
}
func (s *sqlStore) Create(id model.FirmwareKey, reader io.Reader) (string, error) {
	buf, err := ioutil.ReadAll(reader)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	h.Write(buf)
	sha := hex.EncodeToString(h.Sum(nil))
	_, err = s.createStmt.Exec(id, buf)
	if err != nil {
		logging.Warning("Unable to store image with id %s: %v", id, err)
	}
	return sha, err

}

func (s *sqlStore) Retrieve(id model.FirmwareKey) (io.ReadCloser, error) {
	buf := make([]byte, MaxImageSize)
	if err := s.retrieveStmt.QueryRow(id).Scan(&buf); err != nil {
		return nil, err
	}
	return ioutil.NopCloser(bytes.NewReader(buf)), nil
}

func (s *sqlStore) Delete(id model.FirmwareKey) error {
	res, err := s.deleteStmt.Exec(id)
	if err != nil {
		if strings.Contains(err.Error(), "constraint") {
			return storage.ErrReference
		}
		logging.Warning("Error removing firmware: %v", err)
		return err
	}
	if count, err := res.RowsAffected(); err != nil || count == 0 {
		return storage.ErrNotFound
	}
	return nil
}
