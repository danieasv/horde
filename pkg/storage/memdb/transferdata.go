package memdb

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/ExploratoryEngineering/logging"
)

// Functions to transplant data from one (SQL) database to another.

// MoveData moves data from one database to another. Tables must exist in both
// source and destination. This is not thread safe and should not be performed
// on a live destination database. Use the clause string to limit the selection.
// The clause is concatenated at the end of the SELECT so it's a prime candidate
// for SQL injection. DO NOT USE THIS in external APIs. Ever.
func MoveData(table string, source, destination *sql.DB, clause string) error {

	queryStmt := fmt.Sprintf("SELECT * FROM %s", table)
	if clause != "" {
		queryStmt = queryStmt + " WHERE " + clause
	}
	rows, err := source.Query(queryStmt)
	if err != nil {
		return err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return err
	}

	fields := strings.Join(cols, ", ")
	var args []string
	for i := range cols {
		args = append(args, fmt.Sprintf("$%d", i+1))
	}

	params := strings.Join(args, ", ")
	insertStmt := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, fields, params)
	destInsert, err := destination.Prepare(insertStmt)
	if err != nil {
		return err
	}
	vals := make([]interface{}, len(cols))
	for i := range vals {
		vals[i] = new(interface{})
	}
	timeToRead := 0.0
	timeToWrite := 0.0

	num := 0
	destination.Exec("BEGIN TRANSACTION")

	for rows.Next() {
		start := time.Now()
		if err = rows.Scan(vals...); err != nil {
			return err
		}
		end := time.Now()
		timeToRead += float64(end.Sub(start)) / float64(time.Microsecond)
		start = time.Now()
		if _, err := destInsert.Exec(vals...); err != nil {
			return err
		}
		end = time.Now()
		timeToWrite += float64(end.Sub(start)) / float64(time.Microsecond)
		num++

		if num%10000 == 0 {
			destination.Exec("END TRANSACTION")
			destination.Exec("BEGIN TRANSACTION")
		}
	}
	destination.Exec("END TRANSACTION")

	logging.Info("Time to read %s (%d rows): %6.3f us (%6.3f us/row) Time to write %6.3f us (%6.3f us/row)",
		table, num,
		timeToRead, timeToRead/float64(num),
		timeToWrite, timeToWrite/float64(num))
	return nil
}
