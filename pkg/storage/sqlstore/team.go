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

type teamStatements struct {
	list            *sql.Stmt
	create          *sql.Stmt
	createMember    *sql.Stmt
	retrieve        *sql.Stmt
	delete          *sql.Stmt
	updateTag       *sql.Stmt
	retrieveMembers *sql.Stmt
	deleteMember    *sql.Stmt
	updateMember    *sql.Stmt
	retriveTag      *sql.Stmt
}

func (s *sqlStore) initTeamStatements() error {
	var err error
	if s.teamStatements.list, err = s.db.Prepare(`
		SELECT
			t.team_id, t.tags,	m.user_id, m.role_id, h.name, h.email,
			h.phone, h.verified_email, h.verified_phone, h.external_id, h.auth_type, h.avatar_url
		FROM team t
		LEFT OUTER JOIN member m ON t.team_id = m.team_id, member m2, hordeuser h
			WHERE m2.user_id = h.user_id AND
				t.team_id = m2.team_id AND
				m2.user_id = $1
		ORDER BY t.team_id, m.user_id
	`); err != nil {
		return err
	}
	if s.teamStatements.create, err = s.db.Prepare(`
		INSERT INTO team (team_id, tags)
			VALUES ($1, $2)`); err != nil {
		return err
	}
	if s.teamStatements.createMember, err = s.db.Prepare(`
		INSERT INTO member (team_id, user_id, role_id)
			VALUES ($1, $2, $3)
	`); err != nil {
		return err
	}
	if s.teamStatements.retrieve, err = s.db.Prepare(`
		SELECT t.team_id, t.tags, m.user_id, m.role_id, h.name, h.email,
			h.phone, h.verified_email, h.verified_phone, h.external_id, h.auth_type, h.avatar_url
		FROM team t
			LEFT OUTER JOIN member m
				ON t.team_id = m.team_id
			LEFT OUTER JOIN hordeuser h
				ON m.user_id = h.user_id
		WHERE t.team_id IN (
			SELECT DISTINCT team_id
			FROM member
			WHERE user_id = $1 AND team_id = $2) ORDER BY h.name
	`); err != nil {
		return err
	}
	if s.teamStatements.delete, err = s.db.Prepare(`
		DELETE FROM team
			WHERE team_id = $1`); err != nil {
		return err
	}

	if s.teamStatements.updateTag, err = s.db.Prepare(`
		UPDATE team
			SET tags = $1
			WHERE team_id IN (
				SELECT team_id
					FROM member
					WHERE user_id = $2 AND
						team_id = $3 AND
						role_id = 1)`); err != nil {
		return err
	}
	if s.teamStatements.retrieveMembers, err = s.db.Prepare(`
		SELECT user_id, role_id
			FROM member
			WHERE team_id = $1`); err != nil {
		return err
	}
	if s.teamStatements.deleteMember, err = s.db.Prepare(`
		DELETE
			FROM member
			WHERE team_id = $1 AND
				user_id = $2
	`); err != nil {
		return err
	}
	if s.teamStatements.updateMember, err = s.db.Prepare(`
		UPDATE member
			SET role_id = $1
			WHERE team_id = $2 AND
				user_id = $3
	`); err != nil {
		return err
	}
	if s.teamStatements.retriveTag, err = s.db.Prepare(`
		SELECT t.tags
			FROM team t, member m
			WHERE t.team_id = m.team_id AND
				t.team_id = $1 AND
				m.user_id = $2`); err != nil {
		return err
	}
	return nil
}

func (s *sqlStore) ListTeams(userID model.UserKey) ([]model.Team, error) {
	var ret []model.Team

	// It might look a bit weird with *two* joins to the member table but
	// one is used for checking membership (m2) and the other is used to
	// retreive team members.
	rows, err := s.teamStatements.list.Query(userID)
	if err == sql.ErrNoRows {
		return ret, nil
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	currentTeam := model.NewTeam()
	currentTeam.ID = model.TeamKey(0)
	for rows.Next() {
		var teamID model.TeamKey
		var member model.Member
		var tags model.Tags
		if err := rows.Scan(&teamID, &tags.TagMap, &member.User.ID, &member.Role, &member.User.Name,
			&member.User.Email, &member.User.Phone, &member.User.VerifiedEmail,
			&member.User.VerifiedPhone, &member.User.ExternalID, &member.User.AuthType, &member.User.AvatarURL); err != nil {
			if err == sql.ErrNoRows {
				// OK - no more rows
				return nil, storage.ErrNotFound
			}
			return ret, err
		}
		if currentTeam.ID == model.TeamKey(0) {
			currentTeam.ID = teamID
			currentTeam.Tags = tags
		}
		if teamID != currentTeam.ID {
			// A new team. Put the old one on the list
			ret = append(ret, currentTeam)
			currentTeam = model.NewTeam()
			currentTeam.ID = teamID
			currentTeam.Tags = tags
		}
		currentTeam.AddMember(member)

	}
	ret = append(ret, currentTeam)
	return ret, nil
}

func (s *sqlStore) NewTeamID() model.TeamKey {
	return model.TeamKey(s.teamKeyGen.NewID())
}

func (s *sqlStore) CreateTeam(team model.Team) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Stmt(s.teamStatements.create).Exec(team.ID, team.TagMap); err != nil {
		tx.Rollback()
		logging.Warning("Unable to insert team: %v", err)
		if strings.Contains(err.Error(), "constraint") {
			return storage.ErrAlreadyExists
		}
		return err
	}
	for _, v := range team.Members {
		if _, err := tx.Stmt(s.teamStatements.createMember).Exec(team.ID, v.User.ID, v.Role); err != nil {
			tx.Rollback()
			logging.Error("Unable to write team member %+v: %v", v, err)
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		logging.Error("Unable to commit new team: %v", err)
		return err
	}
	return nil
}

func (s *sqlStore) RetrieveTeam(userID model.UserKey, teamID model.TeamKey) (model.Team, error) {
	var team model.Team

	rows, err := s.teamStatements.retrieve.Query(userID, teamID)
	if err != nil {
		return model.Team{}, err
	}

	defer rows.Close()

	for rows.Next() {
		var teamID model.TeamKey
		var member model.Member
		var tags model.Tags
		err := rows.Scan(&teamID, &tags.TagMap, &member.User.ID, &member.Role, &member.User.Name,
			&member.User.Email, &member.User.Phone, &member.User.VerifiedEmail, &member.User.VerifiedPhone, &member.User.ExternalID, &member.User.AuthType, &member.User.AvatarURL)
		if err == sql.ErrNoRows {
			return model.Team{}, storage.ErrNotFound
		}
		if team.ID == model.TeamKey(0) {
			team.ID = teamID
			team.Tags = tags
		}
		team.AddMember(member)
	}
	if team.ID == model.TeamKey(0) {
		return team, storage.ErrNotFound
	}
	return team, nil
}

func (s *sqlStore) DeleteTeam(userID model.UserKey, teamID model.TeamKey) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if err := s.utils.EnsureAdminOfTeam(tx, userID, teamID); err != nil {
		tx.Rollback()
		return err
	}
	if _, err := tx.Stmt(s.teamStatements.delete).Exec(teamID); err != nil {
		tx.Rollback()
		if strings.Contains(err.Error(), "constraint") {
			return storage.ErrReference
		}
		return err
	}
	tx.Commit()
	return nil
}

func (s *sqlStore) UpdateTeam(userID model.UserKey, team model.Team) error {
	tx, err := s.db.Begin()
	if err != nil {
		logging.Warning("Unable to create new transaction for team update: %v", err)
		return err
	}
	if err := s.utils.EnsureAdminOfTeam(tx, userID, team.ID); err != nil {
		tx.Rollback()
		return err
	}
	res, err := tx.Stmt(s.teamStatements.updateTag).Exec(team.TagMap, userID, team.ID)
	if err != nil {
		tx.Rollback()
		logging.Warning("Unable to update team: %v", err)
		if strings.Contains(err.Error(), "constraint") {
			return storage.ErrAlreadyExists
		}
		return err
	}
	if count, err := res.RowsAffected(); err != nil || count == 0 {
		tx.Rollback()
		return storage.ErrNotFound
	}

	// Retrieve list of members
	rows, err := tx.Stmt(s.teamStatements.retrieveMembers).Query(team.ID)
	if err != nil {
		tx.Rollback()
		logging.Warning("Unable to retrieve member list for team: %v", err)
		return err
	}

	defer rows.Close()

	var existingMembers []model.Member
	for rows.Next() {
		member := model.Member{}
		if err := rows.Scan(&member.User.ID, &member.Role); err != nil {
			tx.Rollback()
			return err
		}
		existingMembers = append(existingMembers, member)
	}

	updatedMembers := make(map[model.UserKey]model.RoleID)
	for _, v := range team.Members {
		updatedMembers[v.User.ID] = v.Role
		existing := false
		for _, ex := range existingMembers {
			if ex.User.ID == v.User.ID {
				existing = true
				break
			}
		}
		if !existing {
			// add member
			if _, err := tx.Stmt(s.teamStatements.createMember).Exec(team.ID, v.User.ID, v.Role); err != nil {
				tx.Rollback()
				logging.Warning("Unable to update member list: %v", err)
				return err
			}
		}
	}
	// Find out what to remove, add and update
	for _, v := range existingMembers {
		role, ok := updatedMembers[v.User.ID]
		if !ok {
			// Doesn't exist, remove it
			if _, err := tx.Stmt(s.teamStatements.deleteMember).Exec(team.ID, v.User.ID); err != nil {
				tx.Rollback()
				logging.Warning("Unable to remove member from member list: %v", err)
				return err
			}
			continue
		}
		if role != v.Role {
			if _, err := tx.Stmt(s.teamStatements.updateMember).Exec(role, team.ID, v.User.ID); err != nil {
				tx.Rollback()
				logging.Warning("Unable to update role in member list: %v", err)
				return err
			}
		}
	}
	if err := tx.Commit(); err != nil {
		logging.Warning("Unable to commit changes: %v", err)
		return err
	}
	return nil
}

func (s *sqlStore) RetrieveTeamTags(userID model.UserKey, teamID string) (model.Tags, error) {
	k, err := model.NewTeamKeyFromString(teamID)
	if err != nil {
		return model.Tags{}, storage.ErrNotFound
	}
	ret := model.NewTags()
	if err := s.teamStatements.retriveTag.QueryRow(k, userID).Scan(&ret.TagMap); err != nil {
		if err == sql.ErrNoRows {
			return ret, storage.ErrNotFound
		}
		return ret, err
	}
	return ret, nil
}

func (s *sqlStore) UpdateTeamTags(userID model.UserKey, teamID string, tags model.Tags) error {
	k, err := model.NewTeamKeyFromString(teamID)
	if err != nil {
		return storage.ErrNotFound
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	res, err := tx.Stmt(s.teamStatements.updateTag).Exec(tags.TagMap, userID, k)
	if err != nil {
		tx.Rollback()
		return err
	}
	if count, err := res.RowsAffected(); err != nil {
		tx.Rollback()
		return err
	} else if count == 0 {
		err := s.utils.EnsureAdminOfTeam(tx, userID, k)
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}
