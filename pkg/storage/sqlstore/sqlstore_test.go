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
	"os"
	"testing"

	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
	"github.com/eesrc/horde/pkg/storage/storetest"
)

const dataCenterID uint8 = 1
const workerID uint16 = 1

func TestSQLiteStore(t *testing.T) {
	s, err := NewSQLMutexStore("sqlite3", ":memory:?_foreign_keys=1", true, dataCenterID, workerID)
	if err != nil {
		t.Fatal(err)
	}
	storetest.StorageTest(t, s)
	storetest.SequenceTest(t, s)
}

func TestPostgreSQLStore(t *testing.T) {
	pg := os.Getenv("POSTGRES")
	if pg == "" {
		t.Log("PostgreSQL test skipped")
		return
	}
	s, err := NewSQLStore("postgres", pg, true, dataCenterID, workerID)
	if err != nil {
		t.Fatal(err)
	}
	storetest.StorageTest(t, s)
	storetest.SequenceTest(t, s)
}

func TestSQLiteAPNStore(t *testing.T) {
	s, err := NewSQLAPNStore("sqlite3", "apnstore.sqlite3?_foreign_keys=1", true)
	if err != nil {
		t.Fatal("Unable to create store: ", err)
	}
	defer os.Remove("apnstore.sqlite3")
	s.RemoveNAS(0, 0)
	s.RemoveAPN(0)

	storetest.TestAPNStore(s, t)
}

func benchmarkTokens(b *testing.B, store storage.DataStore, tokens []string) {
	l := len(tokens)
	for i := 0; i < b.N; i++ {
		store.RetrieveToken(tokens[i%l])
	}
}

func fetchExistingTokens(driver, connStr string) ([]string, error) {
	db, err := sql.Open(driver, connStr)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.Query("SELECT token FROM token LIMIT 4000")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tokens []string
	for rows.Next() {
		token := ""

		if err := rows.Scan(&token); err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}
	return tokens, nil
}

func BenchmarkSQLStoreTokens(b *testing.B) {
	connStr := "postgres://localhost/horde?sslmode=disable"
	driver := "postgres"

	s, err := NewSQLStore(driver, connStr, false, dataCenterID, workerID)
	if err != nil {
		b.Fatal(err)
	}
	tokens, err := fetchExistingTokens(driver, connStr)
	if err != nil {
		b.Fatal(err)
	}

	// Start-up is expensive
	b.ResetTimer()

	benchmarkTokens(b, s, tokens)
}

func benchmarkTokenList(b *testing.B, store storage.DataStore, users []model.UserKey) {
	l := len(users)
	for i := 0; i < b.N; i++ {
		store.ListTokens(users[i%l])
	}
}
func fetchExistingTokenUsers(driver, connStr string) ([]model.UserKey, error) {
	db, err := sql.Open(driver, connStr)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.Query("SELECT DISTINCT user_id FROM token LIMIT 4000")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []model.UserKey
	for rows.Next() {
		var uid model.UserKey

		if err := rows.Scan(&uid); err != nil {
			return nil, err
		}
		users = append(users, uid)
	}
	return users, nil
}
func BenchmarkSQLStoreTokenList(b *testing.B) {
	connStr := "postgres://localhost/horde?sslmode=disable"
	driver := "postgres"

	s, err := NewSQLStore(driver, connStr, false, dataCenterID, workerID)
	if err != nil {
		b.Fatal(err)
	}
	users, err := fetchExistingTokenUsers(driver, connStr)
	if err != nil {
		b.Fatal(err)
	}

	// Start-up is expensive
	b.ResetTimer()

	benchmarkTokenList(b, s, users)
}

type userTeam struct {
	userID model.UserKey
	teamID model.TeamKey
}

func benchmarkTeamList(b *testing.B, store storage.DataStore, users []userTeam) {
	for i := 0; i < b.N; i++ {
		if _, err := store.ListTeams(users[i%len(users)].userID); err != nil {
			b.Fatal(err)
		}
	}
}

func fetchExistingTeamUsers(driver, connStr string) ([]userTeam, error) {
	db, err := sql.Open(driver, connStr)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.Query("SELECT DISTINCT team_id, user_id FROM member LIMIT 4000")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []userTeam
	for rows.Next() {
		var tu userTeam

		if err := rows.Scan(&tu.teamID, &tu.userID); err != nil {
			return nil, err
		}
		users = append(users, tu)
	}
	return users, nil
}

func BenchmarkSQLTeamList(b *testing.B) {
	connStr := "postgres://localhost/horde?sslmode=disable"
	driver := "postgres"

	s, err := NewSQLStore(driver, connStr, false, dataCenterID, workerID)
	if err != nil {
		b.Fatal(err)
	}
	users, err := fetchExistingTeamUsers(driver, connStr)
	if err != nil {
		b.Fatal(err)
	}

	// Start-up is expensive
	b.ResetTimer()

	benchmarkTeamList(b, s, users)
}

func benchmarkTeam(b *testing.B, store storage.DataStore, teamUsers []userTeam) {
	for i := 0; i < b.N; i++ {
		ut := teamUsers[i%len(teamUsers)]
		if _, err := store.RetrieveTeam(ut.userID, ut.teamID); err != nil {
			b.Fatalf("Can't find team/user combo %+v: %v", ut, err)
		}
	}
}

func BenchmarkSQLTeam(b *testing.B) {
	connStr := "postgres://localhost/horde?sslmode=disable"
	driver := "postgres"

	s, err := NewSQLStore(driver, connStr, false, dataCenterID, workerID)
	if err != nil {
		b.Fatal(err)
	}
	users, err := fetchExistingTeamUsers(driver, connStr)
	if err != nil {
		b.Fatal(err)
	}

	// Start-up is expensive
	b.ResetTimer()

	benchmarkTeam(b, s, users)
}
