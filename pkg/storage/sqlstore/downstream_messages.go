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
	"sync"
	"time"

	"github.com/ExploratoryEngineering/logging"
	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"

	// SQLite3 driver for testing, local instances and in-memory database
	_ "github.com/mattn/go-sqlite3"
	//PostgreSQL driver for production servers and Real Backends (tm)
	_ "github.com/lib/pq"
)

const downstreamStoreSchema = `
		CREATE TABLE IF NOT EXISTS downstream_messages (
			message_id BIGINT       NOT NULL,
			apn_id     INT          NOT NULL,
			nas_id     INT          NOT NULL,
			transport  INT          NOT NULL,
			created    BIGINT       NOT NULL,
			device_id  BIGINT       NOT NULL,
			message    BYTES        NOT NULL,
			CONSTRAINT downstream_pk PRIMARY KEY (message_id));
		CREATE INDEX IF NOT EXISTS downstream_deviceid ON downstream_messages(device_id);
		CREATE INDEX IF NOT EXISTS downstream_transport ON downstream_messages(transport);
		CREATE INDEX IF NOT EXISTS downstream_apnid ON downstream_messages(apn_id);
		CREATE INDEX IF NOT EXISTS downstream_nasid ON downstream_messages(nas_id);
		CREATE INDEX IF NOT EXISTS downstream_created ON downstream_messages(created);
	`

// NewDownstreamStore returns a new downstream message store. The SQL parameters
// are typically the same as for the regular backend store.
func NewDownstreamStore(kg storage.SequenceStore, params Parameters, dcID uint8, workerID uint16) (storage.DownstreamStore, error) {
	ret := &downstreamStore{
		outQueue:   make([]outMsg, 0),
		inProgress: make([]outMsg, 0),
		mutex:      &sync.Mutex{},
	}
	var err error
	ret.db, err = sql.Open(params.Type, params.ConnectionString)
	if err != nil {
		return nil, err
	}
	if params.CreateSchema {
		schema := NewSchema(params.Type, downstreamStoreSchema)
		if err := schema.Create(ret.db); err != nil {
			return nil, err
		}
	}
	if err := ret.prepareStatements(); err != nil {
		return nil, err
	}
	// Pull the list of messages from the backend store and queue into channel
	if err := ret.loadList(); err != nil {
		return nil, err
	}
	ret.keyGenerator = storage.NewKeyGenerator(dcID, workerID, downMessageIDSequenceName, kg)
	ret.keyGenerator.Start()
	return ret, nil
}

const downMessageIDSequenceName = "messageid"

type outMsg struct {
	ID        model.MessageKey
	DeviceID  model.DeviceKey
	Transport model.MessageTransport
	Payload   []byte
	ApnID     int
	NasID     int
}
type downstreamStore struct {
	keyGenerator *storage.KeyGenerator
	db           *sql.DB
	outQueue     []outMsg
	inProgress   []outMsg
	mutex        *sync.Mutex
	create       *sql.Stmt
	delete       *sql.Stmt
}

func (m *downstreamStore) loadList() error {
	rows, err := m.db.Query(`SELECT message_id, apn_id, nas_id, transport, device_id, message
		FROM downstream_messages ORDER BY created ASC`)
	if err != nil {
		return err
	}
	defer rows.Close()
	m.mutex.Lock()
	defer m.mutex.Unlock()
	for rows.Next() {
		msg := outMsg{}
		if err := rows.Scan(&msg.ID, &msg.ApnID, &msg.NasID, &msg.Transport, &msg.DeviceID, &msg.Payload); err != nil {
			return err
		}
		m.outQueue = append(m.outQueue, msg)
	}
	return nil
}
func (m *downstreamStore) prepareStatements() error {
	var err error
	if m.create, err = m.db.Prepare(`
		INSERT INTO downstream_messages (message_id, apn_id, nas_id, transport, created, device_id, message)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`); err != nil {
		return err
	}
	if m.delete, err = m.db.Prepare(`
		DELETE FROM downstream_messages WHERE message_id = $1
	`); err != nil {
		return err
	}

	return nil
}

func (m *downstreamStore) NewMessageID() model.MessageKey {
	return model.MessageKey(m.keyGenerator.NewID())
}

func (m *downstreamStore) Create(apnID int, nasID int, deviceID model.DeviceKey, id model.MessageKey, transport model.MessageTransport, message []byte) error {
	// Store in database, append to queue
	res, err := m.create.Exec(id, apnID, nasID, transport, time.Now().UnixNano(), deviceID, message)
	if err != nil {
		if strings.Index(err.Error(), "constraint") > 0 {
			return storage.ErrAlreadyExists
		}
		logging.Warning("Unable to create downstream message %s: %v", id.String(), err)
		return storage.ErrInternal
	}
	if n, err := res.RowsAffected(); err != nil || n == 0 {
		logging.Warning("Error inserting row into downstream queue: %v (%d rows inserted)", err, n)
		return storage.ErrInternal
	}
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.outQueue = append(m.outQueue, outMsg{
		ID:        id,
		ApnID:     apnID,
		NasID:     nasID,
		Transport: transport,
		Payload:   message[:],
		DeviceID:  deviceID,
	})
	return nil
}

func (m *downstreamStore) Retrieve(apnID int, nasID int, transport model.MessageTransport) (model.MessageKey, []byte, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	for i, v := range m.outQueue {
		if v.ApnID == apnID && v.NasID == nasID && v.Transport == transport {
			ret := m.outQueue[i]
			m.outQueue = append(m.outQueue[:i], m.outQueue[i+1:]...)
			m.inProgress = append(m.inProgress, ret)
			return ret.ID, ret.Payload, nil
		}
	}
	return 0, nil, storage.ErrNotFound
}

func (m *downstreamStore) RetrieveByDevice(deviceID model.DeviceKey, transport model.MessageTransport) (model.MessageKey, []byte, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for i, v := range m.outQueue {
		if v.DeviceID == deviceID && v.Transport == transport {
			ret := m.outQueue[i]
			m.outQueue = append(m.outQueue[:i], m.outQueue[i+1:]...)
			m.inProgress = append(m.inProgress, ret)
			return ret.ID, ret.Payload, nil
		}
	}
	return 0, nil, storage.ErrNotFound
}

func (m *downstreamStore) Delete(id model.MessageKey) error {
	res, err := m.delete.Exec(id)
	if err != nil {
		logging.Warning("Unable to remove downstream message with ID %d: %v", id, err)
		return err
	}
	if n, err := res.RowsAffected(); err != nil || n == 0 {
		return storage.ErrNotFound
	}
	m.mutex.Lock()
	defer m.mutex.Unlock()
	for i, v := range m.inProgress {
		if v.ID == id {
			m.inProgress = append(m.inProgress[:i], m.inProgress[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *downstreamStore) Release(id model.MessageKey) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	for i, v := range m.inProgress {
		if v.ID == id {
			m.outQueue = append(m.outQueue, m.inProgress[i])
			m.inProgress = append(m.inProgress[:i], m.inProgress[i+1:]...)
			return
		}
	}
}
