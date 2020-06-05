package deviceio

//
// Copyright 2020 Telenor Digital AS
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
import (
	"database/sql"
	"sync/atomic"
	"time"

	"github.com/ExploratoryEngineering/logging"
	"github.com/golang/protobuf/proto"
	_ "github.com/mattn/go-sqlite3" // use sqlite driver
)

const backlogUDPDatabase = "backlog-udp.db"
const backlogCoAPDatabase = "backlog-coap.db"

// messageBacklog is a backlog for messages stored locally. The messages are
// put into the backlog when they arrive and removed when the upstream sink
// has confirmed receiption of the message. Messages can be retrieved via the
// message channel. Messages received will be flagged as in transit. If the
// redelivery of the message succeeds it can be removed from the backlog
type messageBacklog interface {
	// Reset loads the backlog from disk, typically on start-up. Any
	Reset() error

	// Add message to the backlog. The message gets an identifier here and is
	// considered "retrieved", ie nobody will get a copy of this.
	Add(data *upstreamData) error

	// Get returns an unconfirmed message from the backlog. If the block
	// flag is set it will block until a message is available.
	Get(block bool) *upstreamData

	// ConfirmRemove removes the message from the backlog permanently. This means
	// that the message is processed successfully.
	ConfirmRemove(data *upstreamData)

	// CancelRemove sets the status flag as not retrieved and is currently not
	// handled.
	CancelRemove(data *upstreamData)
}

const createMessageTable = `
CREATE TABLE IF NOT EXISTS backlog (
	msgid BIGINT NOT NULL,
	msg BLOB NOT NULL,
	retrieved BOOL NOT NULL
)
`
const createMessageIndex = `CREATE INDEX IF NOT EXISTS backlog_pk ON backlog(msgid)`
const insertQuery = `
	INSERT INTO backlog (
		msgid,
		msg,
		retrieved)
	VALUES ($1, $2, 1)`

const removeQuery = `DELETE FROM backlog WHERE msgid = $1`
const updateQuery = `UPDATE BACKLOG SET retrieved = $1 WHERE msgid = $2`
const selectQuery = `
	SELECT
		msg
	FROM
		backlog
	WHERE
		retrieved = 0
	ORDER BY
		msgid DESC
	LIMIT 1
`

const resetQuery = `
		UPDATE backlog
			SET retrieved = 0
			WHERE retrieved = 1
`
const countQuery = `SELECT COUNT(*) FROM backlog`

// Wait time before sqlite flushes to disk. Message processing have priority
// over committing changes. When there's commitInterval idle time the
// transactions are written to disk
const commitInterval = 50 * time.Millisecond

// Number of queries before starting a new transaction. This is checked while
// processing messages; after flushCount write operations sqlite writes the
// transactions to disk. Enlarge to reduce number of disk operations.
const flushCount = 1000

type backlog struct {
	sequence      *int64
	db            *sql.DB
	backlogChan   chan upstreamData
	appendChan    chan upstreamData
	confirmChan   chan int64
	cancelChan    chan int64
	wantBacklog   chan bool
	selectStmt    *sql.Stmt // Select next message
	insertStmt    *sql.Stmt // Add message
	removeStmt    *sql.Stmt // Confirmed remove
	updateStmt    *sql.Stmt // Retrieve
	commitCounter int
}

const maxWaitQueue = 10

func newMessageBacklog(database string) (messageBacklog, error) {
	db, err := sql.Open("sqlite3", database)
	if err != nil {
		return nil, err
	}
	ret := &backlog{
		db:          db,
		sequence:    new(int64),
		backlogChan: make(chan upstreamData),
		appendChan:  make(chan upstreamData),
		confirmChan: make(chan int64),
		cancelChan:  make(chan int64),
		wantBacklog: make(chan bool, maxWaitQueue),
	}
	if err := ret.setup(); err != nil {
		return nil, err
	}

	go ret.processingLoop()
	return ret, nil
}

func (b *backlog) setup() error {
	var err error
	_, err = b.db.Exec(createMessageTable)
	if err != nil {
		logging.Error("Could not create backlog table: %v", err)
		return err
	}
	_, err = b.db.Exec(createMessageIndex)
	if err != nil {
		return err
	}
	b.selectStmt, err = b.db.Prepare(selectQuery)
	if err != nil {
		return err
	}
	b.insertStmt, err = b.db.Prepare(insertQuery)
	if err != nil {
		return err
	}
	b.removeStmt, err = b.db.Prepare(removeQuery)
	if err != nil {
		return err
	}
	b.updateStmt, err = b.db.Prepare(updateQuery)
	if err != nil {
		return err
	}

	b.db.Exec("BEGIN TRANSACTION")
	return nil
}

func (b *backlog) maybeCommit() {
	b.commitCounter++
	if b.commitCounter > flushCount {
		b.commitCounter = 0
		b.commitChanges()
	}
}

// Run a loop in the background to avoid race conditions on SQLite
func (b *backlog) processingLoop() {
	workBuf := make([]byte, 4096)
	for {
		select {
		case m := <-b.appendChan:
			buf, err := proto.Marshal(&m.Msg)
			if err != nil {
				panic(err.Error())
			}
			_, err = b.insertStmt.Exec(m.Msg.Id, buf)
			if err != nil {
				logging.Error("Got error inserting message: %v. Stopping", err)
				return
			}
			b.maybeCommit()

		case id := <-b.confirmChan:
			_, err := b.removeStmt.Exec(id)
			if err != nil {
				logging.Error("Got error removing message: %v. Stopping", err)
				return
			}
			b.maybeCommit()

		case id := <-b.cancelChan:
			// Cancel retrieval - reset flag on message
			_, err := b.updateStmt.Exec(0, id)
			if err != nil {
				logging.Error("Got error updating state from cancel: %v. Stopping.", err)
				return
			}
			b.maybeCommit()

		case <-b.wantBacklog:
			// Pick top message from list, send it
			res := b.selectStmt.QueryRow()
			if err := res.Scan(&workBuf); err != nil {
				if err == sql.ErrNoRows {
					// no more messages
					continue
				}
				logging.Error("Got error scanning for new message:%v. Stopping.", err)
				return
			}
			nextMessage := upstreamData{}
			if err := proto.Unmarshal(workBuf, &nextMessage.Msg); err != nil {
				panic(err.Error())
			}
			select {
			case b.backlogChan <- nextMessage:
				// Sent message - set state to 1
				_, err := b.updateStmt.Exec(1, nextMessage.Msg.Id)
				if err != nil {
					logging.Error("Got error updating state: %v. Stopping.", err)
					return
				}
			default:
				// nobody's reading - continue
			}
		case <-time.After(commitInterval):
			b.commitChanges()
		}
	}
}

func (b *backlog) commitChanges() {
	b.db.Exec("END TRANSACTION")
	b.db.Exec("BEGIN TRANSACTION")
}

func (b *backlog) Reset() error {
	result, err := b.db.Exec(resetQuery)
	if err != nil {
		return err
	}
	pendingNum, err := result.RowsAffected()
	if err != nil {
		return err
	}
	result, err = b.db.Exec(countQuery)
	if err != nil {
		return err
	}
	totalNum, err := result.RowsAffected()
	if err != nil {
		return err
	}

	logging.Info("%d messages in backlog, %d was pending", totalNum, pendingNum)
	return nil
}

func (b *backlog) Add(data *upstreamData) error {
	id := atomic.AddInt64(b.sequence, 1)
	data.Msg.Id = id
	b.appendChan <- *data
	return nil
}

func (b *backlog) Get(blocking bool) *upstreamData {
	if blocking {
		b.wantBacklog <- true
		ret := <-b.backlogChan
		return &ret
	}
	select {
	case b.wantBacklog <- true:
		select {
		case ret := <-b.backlogChan:
			return &ret
		case <-time.After(1 * time.Millisecond):
			// The message won't appear right away on the backlog channel.
			// Wait until returning. This *does* limit the check rate to
			// 1000 messages/second for a single client but the number of
			// clients can be increased to compensate for this.
			return nil

		}
	default:
		return nil
	}
}

func (b *backlog) CancelRemove(data *upstreamData) {
	b.cancelChan <- data.Msg.Id
}

func (b *backlog) ConfirmRemove(data *upstreamData) {
	b.confirmChan <- data.Msg.Id
}
