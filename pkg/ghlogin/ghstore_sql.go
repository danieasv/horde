package ghlogin
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

	// SQLite3 driver for testing, local instances and in-memory database
	_ "github.com/mattn/go-sqlite3"
	//PostgreSQL driver for production servers and Real Backends (tm)
	_ "github.com/lib/pq"
)

// NewSQLSessionStore creates a sql-backed session streo
func NewSQLSessionStore(driver, connectionString string) (SessionStore, error) {
	ret := sqlSessionStore{}

	var err error
	if ret.db, err = sql.Open(driver, connectionString); err != nil {
		return nil, err
	}
	if err := ret.createSchema(); err != nil {
		return nil, err
	}
	if err := ret.init(); err != nil {
		return nil, err
	}
	return &ret, nil
}

type sqlSessionStore struct {
	db              *sql.DB
	createState     *sql.Stmt
	removeState     *sql.Stmt
	createSession   *sql.Stmt
	retrieveSession *sql.Stmt
	removeSession   *sql.Stmt
	listSessions    *sql.Stmt
	refreshSession  *sql.Stmt
}

func (s *sqlSessionStore) createSchema() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS ghstate (
			state          VARCHAR(128)   NOT NULL,
			CONSTRAINT ghstate_pk PRIMARY KEY (state)
		)`)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS ghsession (
			session_id     VARCHAR(32)    NOT NULL,
			access_token   VARCHAR(128)  NOT NULL,
			expires        BIGINT        NOT NULL,
			profile        JSON          NOT NULL,
			CONSTRAINT ghsession_pk PRIMARY KEY (session_id)
		)`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		CREATE INDEX IF NOT EXISTS ghsession_expires ON ghsession(expires)`)
	if err != nil {
		return err
	}

	return nil
}

func (s *sqlSessionStore) init() error {
	var err error
	if s.createState, err = s.db.Prepare(`
		INSERT INTO ghstate (state)
			VALUES ($1)`); err != nil {
		return err
	}
	if s.removeState, err = s.db.Prepare(`
		DELETE FROM ghstate
			WHERE state = $1`); err != nil {
		return err
	}
	if s.createSession, err = s.db.Prepare(`
		INSERT INTO ghsession (session_id, access_token, expires, profile)
			VALUES ($1, $2, $3, $4)
	`); err != nil {
		return err
	}

	if s.retrieveSession, err = s.db.Prepare(`
		SELECT session_id, access_token, expires, profile
			FROM ghsession
			WHERE session_id = $1 AND expires > $2`); err != nil {
		return err
	}
	if s.removeSession, err = s.db.Prepare(`
		DELETE FROM ghsession
			WHERE session_id = $1`); err != nil {
		return err
	}
	if s.listSessions, err = s.db.Prepare(`
		SELECT session_id, access_token, expires, profile
			FROM ghsession WHERE expires < $1`); err != nil {
		return err
	}
	if s.refreshSession, err = s.db.Prepare(`
		UPDATE ghsession
			SET expires = expires + $1
			WHERE session_id = $2`); err != nil {
		return err
	}
	return nil
}
func (s *sqlSessionStore) PutState(state string) error {
	rows, err := s.createState.Exec(state)
	if err != nil {
		return err
	}
	if count, err := rows.RowsAffected(); err != nil || count == 0 {
		if err != nil {
			return err
		}
		return errors.New("state not stored")
	}
	return nil
}

func (s *sqlSessionStore) RemoveState(state string) error {
	rows, err := s.removeState.Exec(state)
	if err != nil {
		return err
	}
	count, err := rows.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return errors.New("state not found")
	}
	return nil
}

func (s *sqlSessionStore) CreateSession(sessionID string, accessToken string, expires int64, profile Profile) error {
	rows, err := s.createSession.Exec(&sessionID, &accessToken, &expires, &profile)
	if err != nil {
		return err
	}
	count, err := rows.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return errors.New("no session created")
	}
	return nil
}

func (s *sqlSessionStore) GetSession(sessionID string, expireTime int64) (Session, error) {
	row := s.retrieveSession.QueryRow(sessionID, expireTime)
	if row == nil {
		return Session{}, errors.New("not found")
	}
	ret := Session{}
	return ret, row.Scan(&ret.SessionID, &ret.AccessToken, &ret.Expires, &ret.Profile)
}

func (s *sqlSessionStore) RemoveSession(sessionID string) error {
	res, err := s.removeSession.Exec(sessionID)
	if err != nil {
		return err
	}
	count, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return errors.New("session not found")
	}
	return nil
}

func (s *sqlSessionStore) GetSessions(time int64) ([]Session, error) {
	rows, err := s.listSessions.Query(time)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ret := make([]Session, 0)
	for rows.Next() {
		sess := Session{}
		if err := rows.Scan(&sess.SessionID, &sess.AccessToken, &sess.Expires, &sess.Profile); err != nil {
			return ret, err
		}
		ret = append(ret, sess)
	}
	return ret, nil

}

func (s *sqlSessionStore) RefreshSession(sessionID string, checkInterval int64) error {
	res, err := s.refreshSession.Exec(checkInterval, sessionID)
	if err != nil {
		return err
	}
	count, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return errors.New("session not found")
	}
	return nil

}
