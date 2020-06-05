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
	"strings"

	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
)

func (s *sqlStore) NewOutputID() model.OutputKey {
	return model.OutputKey(s.outputKeyGen.NewID())
}

type outputStatements struct {
	list                 *sql.Stmt
	collectionMembership *sql.Stmt
	create               *sql.Stmt
	retrieve             *sql.Stmt
	delete               *sql.Stmt
	update               *sql.Stmt
	retrieveTags         *sql.Stmt
	updateTags           *sql.Stmt
	collectionExists     *sql.Stmt
	fullList             *sql.Stmt
}

func (s *sqlStore) initOutputStatements() error {
	var err error
	if s.outputStatements.list, err = s.db.Prepare(`
		SELECT o.output_id, o.output_type, o.collection_id, o.config, o.enabled, o.tags, c.field_mask
			FROM output o, collection c
			WHERE o.collection_id = c.collection_id AND o.output_id IN (
				SELECT o.output_id
				FROM output o, collection c, member m
				WHERE o.collection_id = c.collection_id	AND
					c.collection_id = $1 AND
					c.team_id = m.team_id AND
					m.user_id = $2)`); err != nil {
		return err
	}
	if s.outputStatements.collectionMembership, err = s.db.Prepare(`
		SELECT COUNT(*)
			FROM collection c, member m
			WHERE c.collection_id = $1 AND
				c.team_id = m.team_id AND
				m.user_id = $2`); err != nil {
		return err
	}
	if s.outputStatements.create, err = s.db.Prepare(`
		INSERT INTO output (output_id, collection_id, output_type, config, enabled, tags)
			VALUES ($1, $2, $3, $4, $5, $6)`); err != nil {
		return err
	}
	if s.outputStatements.retrieve, err = s.db.Prepare(`
		SELECT o.output_id, o.collection_id, o.output_type, o.config, o.enabled, o.tags, c.field_mask
			FROM output o, collection c, member m
			WHERE o.output_id = $1 AND o.collection_id = $2 AND
				c.collection_id = o.collection_id AND
				c.team_id = m.team_id AND
				m.user_id = $3`); err != nil {
		return err
	}
	if s.outputStatements.delete, err = s.db.Prepare(`
		DELETE FROM output
			WHERE output_id = $1`); err != nil {
		return err
	}
	if s.outputStatements.update, err = s.db.Prepare(`
		UPDATE output
			SET output_type = $1,
				collection_id = $2,
				config = $3,
				enabled = $4,
				tags = $5
			WHERE output_id = $6`); err != nil {
		return err
	}
	if s.outputStatements.retrieveTags, err = s.db.Prepare(`
		SELECT o.tags
			FROM output o, collection c, member m
			WHERE o.output_id = $1 AND
				o.collection_id = c.collection_id AND
				c.team_id = m.team_id AND
				m.user_id = $2`); err != nil {
		return err
	}
	if s.outputStatements.updateTags, err = s.db.Prepare(`
		UPDATE output
			SET tags = $1
			WHERE output_id IN (
				SELECT o.output_id
				FROM output o, collection c, member m
				WHERE o.output_id = $2 AND
					o.collection_id = c.collection_id AND
					c.team_id = m.team_id AND
					user_id = $3 AND
					role_id = 1)`); err != nil {
		return err
	}
	if s.outputStatements.collectionExists, err = s.db.Prepare(`
		SELECT o.collection_id
			FROM output o
			WHERE o.output_id = $1`); err != nil {
		return err
	}
	if s.outputStatements.fullList, err = s.db.Prepare(`
		SELECT o.output_id, o.collection_id, o.output_type, o.config, o.enabled, o.tags, c.field_mask
			FROM output o, collection c
			WHERE o.collection_id = c.collection_id`); err != nil {
		return err
	}
	return nil
}

func (s *sqlStore) ListOutputs(userID model.UserKey, collectionID model.CollectionKey) ([]model.Output, error) {
	ret := []model.Output{}
	rows, err := s.outputStatements.list.Query(collectionID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var o model.Output
		if err := rows.Scan(&o.ID, &o.Type, &o.CollectionID, &o.Config, &o.Enabled, &o.TagMap, &o.CollectionFieldMask); err != nil {
			if err == sql.ErrNoRows {
				return nil, storage.ErrNotFound
			}
			return nil, err
		}
		ret = append(ret, o)
	}
	// This might be either an empty list or unknown collection. Check if the user is really a member of the team
	if len(ret) == 0 {
		count := 0
		if err := s.outputStatements.collectionMembership.QueryRow(collectionID, userID).Scan(&count); err == sql.ErrNoRows || count == 0 {
			return nil, storage.ErrNotFound
		}
	}
	return ret, nil
}

func (s *sqlStore) CreateOutput(userID model.UserKey, output model.Output) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if _, err := s.utils.EnsureAdminOfCollection(tx, userID, output.CollectionID); err != nil {
		tx.Rollback()
		return err
	}
	_, err = tx.Stmt(s.outputStatements.create).Exec(output.ID, output.CollectionID, output.Type, output.Config, output.Enabled, output.TagMap)

	if err != nil {
		tx.Rollback()
		if strings.Contains(err.Error(), "constraint") {
			return storage.ErrAlreadyExists
		}
		return err
	}
	tx.Commit()
	return nil
}

func (s *sqlStore) RetrieveOutput(userID model.UserKey, collectionID model.CollectionKey, outputID model.OutputKey) (model.Output, error) {
	var ret model.Output
	if err := s.outputStatements.retrieve.QueryRow(outputID, collectionID, userID).Scan(&ret.ID, &ret.CollectionID, &ret.Type, &ret.Config, &ret.Enabled, &ret.TagMap, &ret.CollectionFieldMask); err != nil {
		if err == sql.ErrNoRows {
			return model.Output{}, storage.ErrNotFound
		}
		return model.Output{}, err
	}
	return ret, nil
}

func (s *sqlStore) DeleteOutput(userID model.UserKey, collectionID model.CollectionKey, outputID model.OutputKey) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if err := s.utils.EnsureAdminOfOutput(tx, userID, collectionID, outputID); err != nil {
		tx.Rollback()
		return err
	}
	if _, err := tx.Stmt(s.outputStatements.delete).Exec(outputID); err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func (s *sqlStore) UpdateOutput(userID model.UserKey, collectionID model.CollectionKey, output model.Output) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if err := s.utils.EnsureAdminOfOutput(tx, userID, collectionID, output.ID); err != nil {
		tx.Rollback()
		return err
	}
	if output.CollectionID != collectionID {
		if _, err := s.utils.EnsureAdminOfCollection(tx, userID, output.CollectionID); err != nil {
			tx.Rollback()
			return err
		}
	}
	if _, err := tx.Stmt(s.outputStatements.update).Exec(output.Type, output.CollectionID, output.Config, output.Enabled, output.TagMap, output.ID); err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func (s *sqlStore) RetrieveOutputTags(userID model.UserKey, outputID string) (model.Tags, error) {
	k, err := model.NewOutputKeyFromString(outputID)
	if err != nil {
		return model.Tags{}, storage.ErrNotFound
	}
	ret := model.NewTags()
	if err := s.outputStatements.retrieveTags.QueryRow(k, userID).Scan(&ret.TagMap); err != nil {
		if err == sql.ErrNoRows {
			return ret, storage.ErrNotFound
		}
		return ret, err
	}
	return ret, nil
}

func (s *sqlStore) UpdateOutputTags(userID model.UserKey, outputID string, tags model.Tags) error {
	k, err := model.NewOutputKeyFromString(outputID)
	if err != nil {
		return storage.ErrNotFound
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	res, err := tx.Stmt(s.outputStatements.updateTags).Exec(tags.TagMap, k, userID)
	if err != nil {
		tx.Rollback()
		return err
	}
	if count, err := res.RowsAffected(); err != nil {
		tx.Rollback()
		return err
	} else if count == 0 {
		var collectionID model.CollectionKey
		if err := tx.Stmt(s.outputStatements.collectionExists).QueryRow(k).Scan(&collectionID); err != nil {
			tx.Rollback()
			if err == sql.ErrNoRows {
				return storage.ErrNotFound
			}
			return err
		}
		_, err := s.utils.EnsureAdminOfCollection(tx, userID, collectionID)
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func (s *sqlStore) OutputListAll() ([]model.Output, error) {
	var ret []model.Output
	rows, err := s.outputStatements.fullList.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var o model.Output
		if err := rows.Scan(&o.ID, &o.CollectionID, &o.Type, &o.Config, &o.Enabled, &o.TagMap, &o.CollectionFieldMask); err != nil {
			return nil, err
		}
		ret = append(ret, o)
	}
	return ret, nil
}
