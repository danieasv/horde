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

	"github.com/ExploratoryEngineering/logging"
	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
)

type userStatements struct {
	retrieveByExternalID *sql.Stmt
	create               *sql.Stmt
	createTeam           *sql.Stmt
	createMember         *sql.Stmt
	update               *sql.Stmt
	retrieve             *sql.Stmt
}

func (s *sqlStore) initUserStatements() error {
	var err error
	if s.userStatements.retrieveByExternalID, err = s.db.Prepare(`
		SELECT
			user_id, name, email, phone, external_id, verified_email,
			verified_phone, deleted, private_team_id,
			avatar_url,  auth_type
			FROM hordeuser
			WHERE external_id=$1 AND auth_type = $2`); err != nil {
		return err
	}
	if s.userStatements.create, err = s.db.Prepare(`
		INSERT INTO hordeuser (user_id, name, email, phone, external_id,
				verified_email, verified_phone, deleted, private_team_id,
				avatar_url,  auth_type)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`); err != nil {
		return err
	}
	if s.userStatements.createTeam, err = s.db.Prepare(`
		INSERT INTO team (team_id, tags)
			VALUES ($1, $2)`); err != nil {
		return err
	}
	if s.userStatements.createMember, err = s.db.Prepare(`
		INSERT
			INTO member (team_id, user_id, role_id)
			VALUES ($1, $2, 1)`); err != nil {
		return err
	}
	if s.userStatements.update, err = s.db.Prepare(`
		UPDATE hordeuser
			SET name = $1, email = $2,
				phone = $3, external_id = $4,
				verified_email = $5, verified_phone = $6,
				deleted = $7, avatar_url = $8
			WHERE user_id = $9
		`); err != nil {
		return err
	}
	if s.userStatements.retrieve, err = s.db.Prepare(`
		SELECT
				user_id, name, email, phone, external_id,
				verified_email, verified_phone, deleted, private_team_id,
				avatar_url, auth_type
			FROM hordeuser
			WHERE user_id = $1`); err != nil {
		return err
	}
	return nil
}

func (s *sqlStore) RetrieveUserByExternalID(id string, auth model.AuthMethod) (*model.User, error) {
	var user model.User
	// TODO: pull from storage
	if err := s.userStatements.retrieveByExternalID.QueryRow(id, auth).Scan(
		&user.ID, &user.Name, &user.Email, &user.Phone,
		&user.ExternalID, &user.VerifiedEmail, &user.VerifiedPhone,
		&user.Deleted, &user.PrivateTeamID,
		&user.AvatarURL, &user.AuthType); err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (s *sqlStore) NewUserID() model.UserKey {
	return model.UserKey(s.userKeyGen.NewID())
}

func (s *sqlStore) CreateUser(user model.User, privateTeam model.Team) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Stmt(s.userStatements.create).Exec(
		user.ID, user.Name, user.Email, user.Phone, user.ExternalID,
		user.VerifiedEmail, user.VerifiedPhone, user.Deleted, user.PrivateTeamID,
		user.AvatarURL, user.AuthType)
	if err != nil {
		tx.Rollback()
		logging.Debug("Constraint error inserting user: %v", err)
		if strings.Contains(err.Error(), "constraint") {
			return storage.ErrAlreadyExists
		}
		return err
	}

	_, err = tx.Stmt(s.userStatements.createTeam).Exec(privateTeam.ID, privateTeam.TagMap)
	if err != nil {
		tx.Rollback()
		logging.Debug("Constraint error inserting team: %v", err)
		if strings.Contains(err.Error(), "constraint") {
			return storage.ErrAlreadyExists
		}
		return err
	}

	_, err = tx.Stmt(s.userStatements.createMember).Exec(privateTeam.ID, user.ID)
	if err != nil {
		tx.Rollback()
		logging.Debug("Constraint error inserting member: %v", err)
		if strings.Contains(err.Error(), "constraint") {
			return storage.ErrAlreadyExists
		}
		return err
	}
	tx.Commit()
	return nil
}

func (s *sqlStore) UpdateUser(user *model.User) error {
	result, err := s.userStatements.update.Exec(
		user.Name, user.Email, user.Phone, user.ExternalID,
		user.VerifiedEmail, user.VerifiedPhone, user.Deleted,
		user.AvatarURL, user.ID)
	if count, err := result.RowsAffected(); count == 0 || err != nil {
		return storage.ErrNotFound
	}
	return err
}

func (s *sqlStore) RetrieveUser(key model.UserKey) (model.User, error) {
	var user model.User
	row := s.userStatements.retrieve.QueryRow(key)
	if row == nil {
		return model.User{}, storage.ErrNotFound
	}
	err := row.Scan(
		&user.ID, &user.Name, &user.Email, &user.Phone, &user.ExternalID,
		&user.VerifiedEmail, &user.VerifiedPhone, &user.Deleted, &user.PrivateTeamID,
		&user.AvatarURL, &user.AuthType)
	if err == sql.ErrNoRows {
		return model.User{}, storage.ErrNotFound
	}
	return user, err
}
