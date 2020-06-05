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

type inviteStatements struct {
	list         *sql.Stmt
	create       *sql.Stmt
	delete       *sql.Stmt
	insertMember *sql.Stmt
	retrieve     *sql.Stmt
}

func (s *sqlStore) initInviteStatements() error {
	var err error

	if s.inviteStatements.list, err = s.db.Prepare(`
		SELECT
			code, user_id, team_id, created
			FROM invite
			WHERE team_id = $1
		`); err != nil {
		return err
	}
	if s.inviteStatements.create, err = s.db.Prepare(`
		INSERT
			INTO invite (code, user_id, team_id, created)
            VALUES ($1, $2, $3, $4)
		`); err != nil {
		return err
	}
	if s.inviteStatements.delete, err = s.db.Prepare(`
		DELETE
			FROM invite
			WHERE team_id = $1 AND code = $2`); err != nil {
		return err
	}
	if s.inviteStatements.insertMember, err = s.db.Prepare(`
		INSERT
			INTO member (team_id, user_id, role_id)
			VALUES ($1, $2, 0)`); err != nil {
		return err
	}
	if s.inviteStatements.retrieve, err = s.db.Prepare(`
		SELECT code, team_id, user_id, created
			FROM invite
			WHERE code = $1`); err != nil {
		return err
	}
	return nil
}

func (s *sqlStore) ListInvites(teamID model.TeamKey, userID model.UserKey) ([]model.Invite, error) {
	tx, err := s.db.Begin()
	if err != nil {
		logging.Warning("Unable to create transaction for invite list: %v", err)
		return nil, err
	}
	if err := s.utils.EnsureAdminOfTeam(tx, userID, teamID); err != nil {
		tx.Rollback()
		return nil, err
	}

	invites := make([]model.Invite, 0)
	rows, err := tx.Stmt(s.inviteStatements.list).Query(teamID)
	if err != nil {
		tx.Rollback()
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var invite model.Invite
		if err := rows.Scan(&invite.Code, &invite.UserID, &invite.TeamID, &invite.Created); err != nil {
			tx.Rollback()
			if err == sql.ErrNoRows {
				return nil, storage.ErrNotFound
			}
			return nil, err
		}
		invites = append(invites, invite)
	}
	tx.Commit()
	return invites, nil
}

func (s *sqlStore) CreateInvite(invite model.Invite) error {
	tx, err := s.db.Begin()
	if err != nil {
		logging.Warning("Unable to create new transaction for invite: %v", err)
		return err
	}

	if err := s.utils.EnsureAdminOfTeam(tx, invite.UserID, invite.TeamID); err != nil {
		tx.Rollback()
		return err
	}
	if err := s.utils.EnsureNotPrivateTeam(tx, invite.TeamID); err != nil {
		tx.Rollback()
		return err
	}
	_, err = tx.Stmt(s.inviteStatements.create).Exec(invite.Code, invite.UserID, invite.TeamID, invite.Created)
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

func (s *sqlStore) DeleteInvite(code string, teamID model.TeamKey, userID model.UserKey) error {
	tx, err := s.db.Begin()
	if err != nil {
		logging.Warning("Unable to create a new transaction to delete invite: %v", err)
		return err
	}
	if err := s.utils.EnsureAdminOfTeam(tx, userID, teamID); err != nil {
		tx.Rollback()
		return err
	}

	result, err := tx.Stmt(s.inviteStatements.delete).Exec(teamID, code)
	if err != nil {
		tx.Rollback()
		return err
	}
	if count, err := result.RowsAffected(); err != nil || count == 0 {
		tx.Rollback()
		return storage.ErrNotFound
	}
	tx.Commit()
	return nil
}

func (s *sqlStore) AcceptInvite(invite model.Invite, userID model.UserKey) error {
	tx, err := s.db.Begin()
	if err != nil {
		logging.Warning("Unable to create a new transaction to accept invite: %v", err)
		return err
	}
	if err := s.utils.EnsureAdminOfTeam(tx, invite.UserID, invite.TeamID); err != nil {
		tx.Rollback()
		return err
	}

	result, err := tx.Stmt(s.inviteStatements.delete).Exec(invite.TeamID, invite.Code)
	if err != nil {
		tx.Rollback()
		if err == sql.ErrNoRows {
			return storage.ErrNotFound
		}
		return err
	}
	if count, err := result.RowsAffected(); err != nil || count == 0 {
		tx.Rollback()

		return storage.ErrNotFound
	}

	_, err = tx.Stmt(s.inviteStatements.insertMember).Exec(invite.TeamID, userID)
	if err != nil {
		tx.Rollback()
		if strings.Index(err.Error(), "constraint") > 0 {
			return storage.ErrAlreadyExists
		}
		return err
	}

	tx.Commit()
	return nil
}

func (s *sqlStore) RetrieveInvite(code string) (model.Invite, error) {
	var ret model.Invite
	row := s.inviteStatements.retrieve.QueryRow(code)
	if row == nil {
		return ret, storage.ErrNotFound
	}
	err := row.Scan(&ret.Code, &ret.TeamID, &ret.UserID, &ret.Created)
	if err == sql.ErrNoRows {
		return ret, storage.ErrNotFound
	}
	return ret, err
}
