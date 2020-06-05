package main
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
	"flag"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	// Use the SQLite driver (for testing)
	_ "github.com/mattn/go-sqlite3"
	// Use pgsql driver
	_ "github.com/lib/pq"
)

func main() {
	file := flag.String("file", "", "CSV file with metadata (IMSI, MSISDN, ICC, SIM Type,...")
	hasHeaders := flag.Bool("header", true, "First line of file contains headers")
	connStr := flag.String("connection-string", "", "Connection string for database")
	dbType := flag.String("db", "postgres", "Database type")
	msisdnPrefix := flag.String("msdisdn-prefix", "47", "MSISDN prefix")

	flag.Parse()

	var db *sql.DB
	var err error
	if *dbType == "sqlite3" {
		db, err = sql.Open("sqlite3", *connStr+"?cache=shared&mode=rwc&_foreign_keys=1&_journal_mode=wal")
	}
	if *dbType == "postgres" {
		db, err = sql.Open("postgres", *connStr)
	}
	if err != nil || db == nil {
		fmt.Printf("Unable to connect to db %s/%s: %v", *dbType, *connStr, err)
		return
	}
	if *file == "" {
		fmt.Println("No input file.")
		return
	}

	bytes, err := ioutil.ReadFile(*file)
	if err != nil {
		fmt.Printf("Unable to read file %s: %v", *file, err)
		return
	}

	str := string(bytes)
	lines := strings.Split(str, "\n")

	success := 0
	for i, v := range lines {
		fields := strings.Split(strings.TrimSpace(v), ",")
		if len(fields) < 4 {
			fmt.Printf("Field count is less than 4")
			return
		}
		if i == 0 && *hasHeaders {
			// Skip headers
			continue
		}
		imsiStr := strings.TrimSpace(fields[0])
		msisdn := strings.TrimSpace(fields[1])
		icc := strings.TrimSpace(fields[2])
		simtype := strings.TrimSpace(fields[3])

		if msisdn == "" || imsiStr == "" {
			fmt.Printf("Skipping line %d: missing IMSI and/or MSISDN\n", i)
			continue
		}
		imsi, err := strconv.ParseInt(imsiStr, 10, 64)
		if err != nil {
			fmt.Printf("Invalid IMSI on line %d (%s). Skipping line\n", i, imsiStr)
			continue
		}
		_, err = db.Exec(`
				INSERT INTO device_lookup (imsi, msisdn, icc, simtype)
				VALUES ($1, $2, $3, $4)`, imsi, *msisdnPrefix+msisdn, icc, simtype)
		if err != nil {
			fmt.Printf("Error inserting line %d: %v\n", i, err)
			continue
		}
		success++
	}
	fmt.Printf("%d rows inserted\n", success)
}
