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
	"errors"
	"strings"

	"github.com/ExploratoryEngineering/logging"
	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
)

type tokenStatements struct {
	insert      *sql.Stmt
	list        *sql.Stmt
	update      *sql.Stmt
	delete      *sql.Stmt
	retrieveTag *sql.Stmt
	updateTag   *sql.Stmt
	retrieve    *sql.Stmt
}

func (s *sqlStore) initTokenStatements() error {
	var err error
	if s.tokenStatements.insert, err = s.db.Prepare(`
		INSERT INTO token (
			token, resource, user_id, write, tags)
			VALUES ($1, $2, $3, $4, $5)`); err != nil {
		return err
	}
	if s.tokenStatements.list, err = s.db.Prepare(`
		SELECT
			token, resource, user_id, write, tags
			FROM token
			WHERE user_id = $1`); err != nil {
		return err
	}
	if s.tokenStatements.update, err = s.db.Prepare(`
		UPDATE token
			SET resource = $1, write = $2, tags = $3
			WHERE token = $4`); err != nil {
		return err
	}
	if s.tokenStatements.delete, err = s.db.Prepare(`
		DELETE
			FROM token
			WHERE token = $1 AND user_id = $2`); err != nil {
		return err
	}
	if s.tokenStatements.retrieveTag, err = s.db.Prepare(`
		SELECT tags
			FROM token
			WHERE token = $1 AND user_id = $2`); err != nil {
		return err
	}
	if s.tokenStatements.updateTag, err = s.db.Prepare(`
		UPDATE token
			SET tags = $1
			WHERE user_id = $2 AND token = $3`); err != nil {
		return err
	}
	if s.tokenStatements.retrieve, err = s.db.Prepare(`
		SELECT token, resource, user_id, write, tags
			FROM token
			WHERE token = $1`); err != nil {
		return err
	}
	return nil
}

func (s *sqlStore) CreateToken(token model.Token) error {
	_, err := s.tokenStatements.insert.Exec(
		token.Token, token.Resource, token.UserID, token.Write, token.TagMap)
	if err != nil {
		logging.Warning("Unable to create token: %v", err)
		if strings.Contains(err.Error(), "constraint") {
			return storage.ErrAlreadyExists
		}
	}
	return err
}

func (s *sqlStore) ListTokens(userID model.UserKey) ([]model.Token, error) {
	var tokens []model.Token
	rows, err := s.tokenStatements.list.Query(userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var token model.Token
		if err := rows.Scan(&token.Token, &token.Resource, &token.UserID, &token.Write, &token.TagMap); err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}
	return tokens, nil
}

func (s *sqlStore) UpdateToken(token model.Token) error {
	result, err := s.tokenStatements.update.Exec(token.Resource, token.Write, token.TagMap, token.Token)
	if err != nil {
		return err
	}
	if count, err := result.RowsAffected(); err != nil || count == 0 {
		return storage.ErrNotFound
	}
	return nil
}

func (s *sqlStore) DeleteToken(userID model.UserKey, token string) error {
	result, err := s.tokenStatements.delete.Exec(token, userID)
	if err != nil {
		return err
	}
	if count, err := result.RowsAffected(); err != nil || count == 0 {
		return storage.ErrNotFound
	}
	return nil
}

func (s *sqlStore) RetrieveTokenTags(userID model.UserKey, token string) (model.Tags, error) {
	ret := model.NewTags()
	row := s.tokenStatements.retrieveTag.QueryRow(token, userID)
	if row == nil {
		return ret, errors.New("unable to query for tokens")
	}
	err := row.Scan(&ret.TagMap)
	if err == sql.ErrNoRows {
		err = storage.ErrNotFound
	}
	return ret, err
}

func (s *sqlStore) UpdateTokenTags(userID model.UserKey, token string, tags model.Tags) error {
	res, err := s.tokenStatements.updateTag.Exec(tags.TagMap, userID, token)
	if err != nil {
		return err
	}
	if count, err := res.RowsAffected(); err != nil || count == 0 {
		return storage.ErrNotFound
	}
	return nil
}

func (s *sqlStore) RetrieveToken(tokenString string) (model.Token, error) {
	var token model.Token
	if err := s.tokenStatements.retrieve.QueryRow(tokenString).Scan(
		&token.Token, &token.Resource, &token.UserID,
		&token.Write, &token.TagMap); err != nil {
		if err == sql.ErrNoRows {
			return token, storage.ErrNotFound
		}
		if err != nil {
			return token, err
		}
	}
	return token, nil
}
