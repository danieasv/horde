package sqlstore

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
	"database/sql"

	"github.com/ExploratoryEngineering/logging"
	"github.com/eesrc/horde/pkg/storage"
)

type sequenceStatements struct {
	update   *sql.Stmt
	insert   *sql.Stmt
	retrieve *sql.Stmt
}

func (s *sqlStore) initSequenceStatements() error {
	var err error
	if s.sequenceStatements.update, err = s.db.Prepare(`
		UPDATE sequence
			SET counter = $1
			WHERE identifier = $2 AND
				counter = $3
	`); err != nil {
		return err
	}
	if s.sequenceStatements.insert, err = s.db.Prepare(`
		INSERT INTO sequence (identifier, counter)
			VALUES ($1, $2)
	`); err != nil {
		return err
	}
	if s.sequenceStatements.retrieve, err = s.db.Prepare(`
		SELECT counter
			FROM sequence
			WHERE identifier = $1
	`); err != nil {
		return err
	}
	return nil
}

func (s *sqlStore) AllocateSequence(identifier string, current uint64, new uint64) bool {
	tx, err := s.db.Begin()
	if err != nil {
		logging.Error("Unable to start transaction for sequence allocation: %v", err)
		return false
	}
	result, err := tx.Stmt(s.sequenceStatements.update).Exec(new, identifier, current)
	if err != nil {
		tx.Rollback()
		logging.Warning("Unable to update row: %v", err)
		return false
	}
	if count, err := result.RowsAffected(); err != nil || count == 0 {
		// No rows updated; attempt insert. If that fails it's a race condition
		if _, err := tx.Stmt(s.sequenceStatements.insert).Exec(identifier, new); err != nil {
			tx.Rollback()
			logging.Warning("Possible race condition for sequence: %v", err)
			return false
		}
	}
	tx.Commit()
	return true
}

func (s *sqlStore) CurrentSequence(identifier string) (uint64, error) {
	var counter int64
	if err := s.sequenceStatements.retrieve.QueryRow(identifier).Scan(&counter); err != nil {
		if err == sql.ErrNoRows {
			return 0, storage.ErrNotFound
		}
		logging.Error("Error doing query: %v", err)
		return 0, err
	}
	return uint64(counter), nil
}
