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
	"fmt"
	"strings"
)

// Schema contains a database schema.
type Schema struct {
	driver     string
	statements string
}

// DBSchema is just a big schema string - this is split automagically into separate statements
// by the Statements function.
// Data types are converted on the fly for SQLite3
const DBSchema = `
-- The user table. This holds the (little) user information we're keeping
-- for each user. The only authentication scheme we support is the CONNECT ID
-- service.
CREATE TABLE IF NOT EXISTS hordeuser (
	user_id         BIGINT       NOT NULL,               -- the user ID,
	external_id     VARCHAR(128) NOT NULL,               -- External user ID
	name            VARCHAR(128) NULL,                   -- (optional) user name
	email           VARCHAR(128) NULL,                   -- (optional) email
	phone           VARCHAR(46)  NULL,                   -- (optional phone)
	deleted         BOOL         NOT NULL DEFAULT FALSE, -- for tombstoning
	verified_email  BOOL         NOT NULL DEFAULT FALSE,
	verified_phone  BOOL         NOT NULL DEFAULT FALSE,
	private_team_id BIGINT       NOT NULL,               -- the user's private team. TODO: Add reference to team DB
	avatar_url      VARCHAR(255) NULL DEFAULT '',
	auth_type       INT          NOT NULL,
	CONSTRAINT hordeuser_pk PRIMARY KEY (user_id)
);

CREATE INDEX IF NOT EXISTS user_name ON hordeuser (name);
CREATE INDEX IF NOT EXISTS user_email ON hordeuser (email);
CREATE UNIQUE INDEX IF NOT EXISTS user_external ON hordeuser(external_id);

-- Token table. This holds the API token for users. Tokens are generated
-- by a secure random generator inside the service.
CREATE TABLE IF NOT EXISTS token (
	token     VARCHAR(64)   NOT NULL,
	resource  VARCHAR(128)  NOT NULL,
	user_id   BIGINT        NOT NULL REFERENCES hordeuser (user_id) ON DELETE CASCADE,
	write     BOOL          NOT NULL DEFAULT FALSE,
	tags      JSON          NULL,

	CONSTRAINT token_pk PRIMARY KEY (token)
);

CREATE INDEX IF NOT EXISTS token_fk1 ON token (user_id);

-- The identifier sequences. The sequences are grouped by the "identifier" field,
-- ie different sites will have different identifiers on different instances running
-- in different data centers (aka regions in AWS). This table might not hold all of
-- the sequences that are in use, just the ones running on the current instance.
CREATE TABLE IF NOT EXISTS sequence (
	identifier VARCHAR(128) NOT NULL, -- identifier
	counter    BIGINT       NOT NULL, -- The current counter value

	CONSTRAINT sequence_pk PRIMARY KEY (identifier)
);


-- Role table. Each member in a team have a role. The role can be one of the
-- predefined roles.
CREATE TABLE IF NOT EXISTS role (
	role_id BYTE        NOT NULL, --
	name    VARCHAR(20) NOT NULL, --

	CONSTRAINT role_pk PRIMARY KEY (role_id)
);

-- Default roles.
INSERT INTO role (role_id, name) SELECT 1, 'Admin' WHERE NOT EXISTS (SELECT role_id FROM role WHERE role_id = 1);
INSERT INTO role (role_id, name) SELECT 0, 'Member' WHERE NOT EXISTS (SELECT role_id FROM role WHERE role_id = 0);

-- Teams are collections of users. A team may contain one or more members, not
-- zero. By default new users are member of their own team (and nothing else)
CREATE TABLE IF NOT EXISTS team (
	team_id       BIGINT NOT NULL, -- The team ID
	tags          JSON   NULL,

	CONSTRAINT team_pk PRIMARY KEY (team_id)
);

-- Members of teams. A user can only have be a member once of a team.
CREATE TABLE IF NOT EXISTS member (
	team_id BIGINT NOT NULL REFERENCES team (team_id) ON DELETE CASCADE,
	user_id BIGINT NOT NULL REFERENCES hordeuser (user_id) ON DELETE CASCADE,
	role_id BYTE   NOT NULL REFERENCES role (role_id), -- The user's role

	CONSTRAINT member_pk PRIMARY KEY (team_id, user_id)
);

CREATE INDEX IF NOT EXISTS member_fk1 ON member(team_id);
CREATE INDEX IF NOT EXISTS member_fk2 ON member(user_id);
CREATE INDEX IF NOT EXISTS member_fk3 ON member(role_id);


--
-- Firmware image metadata. Firmware images are stored outside of the database.
-- The name of the external resource is inferred from the firmware ID.
CREATE TABLE IF NOT EXISTS firmware (
	firmware_id   BIGINT       NOT NULL,
	filename      VARCHAR(128) NOT NULL,
	version       VARCHAR(128) NOT NULL,
	length        INT          NOT NULL,
	sha256        VARCHAR(64)  NOT NULL,
	created       DATETIME     NOT NULL,
	collection_id BIGINT       NOT NULL, -- constraint is added later
	tags          JSON         NULL,

	CONSTRAINT firmware_pk PRIMARY KEY (firmware_id)
);

CREATE INDEX IF NOT EXISTS firmware_fk1 ON firmware(collection_id);
CREATE UNIQUE INDEX IF NOT EXISTS firmware_sha ON firmware(collection_id, sha256);
CREATE UNIQUE INDEX IF NOT EXISTS firmware_version ON firmware(collection_id, version);

-- Collections of devices. A device will always be in a collection.
CREATE TABLE IF NOT EXISTS collection (
	collection_id      BIGINT  NOT NULL,                            -- Collection ID
	field_mask         INT     NOT NULL DEFAULT 0,                  -- Field masks for devices
	team_id            BIGINT  NOT NULL REFERENCES team (team_id),  -- Collection owner
	fw_current_version BIGINT  NULL     REFERENCES firmware (firmware_id),
	fw_target_version  BIGINT  NULL     REFERENCES firmware (firmware_id),
	fw_management      SMALLINT NOT NULL DEFAULT 32,
	tags               JSON    NULL,                                -- Tags for collection

	CONSTRAINT collection_pk PRIMARY KEY (collection_id)
);

-- This breaks in SQLite
--ALTER TABLE firmware ADD CONSTRAINT firmware_fk1 FOREIGN KEY (collection_id) REFERENCES collection (collection_id);

CREATE INDEX IF NOT EXISTS collection_fk1 ON collection (team_id);

-- The actual devices. Device IDs are unique, as well as IMEI and IMSI.
-- All devices have one owner and are optionally a part of a collection.
-- Note that the APN ID is referenced in the APN table but it might not be
-- used in this database so there's no relation.
CREATE TABLE IF NOT EXISTS device (
	device_id          BIGINT       NOT NULL,
	imei               BIGINT       NOT NULL,
	imsi               BIGINT       NOT NULL,
	collection_id      BIGINT       NOT NULL REFERENCES collection (collection_id),
	tags               JSON         NULL,
	net_apn_id         INT          NULL, -- APN ID for last allocation/message
	net_nas_id         INT          NULL, -- NAS ID for last allocation/message
	net_allocated_ip   VARCHAR(32)  NULL, -- Allocated IP
	net_allocated_at   DATETIME     NULL, -- Allocated time for IP
	net_cell_id        BIGINT       NULL, -- Last reported cell ID
	fw_current_version BIGINT       NULL REFERENCES firmware(firmware_id),
	fw_target_version  BIGINT       NULL REFERENCES firmware(firmware_id),
	fw_serial_number   VARCHAR(64)  NULL,
	fw_model_number    VARCHAR(64)  NULL,
	fw_manufacturer    VARCHAR(64)  NULL,
	fw_version         VARCHAR(64)  NULL,
	fw_state           CHAR(1)      NOT NULL DEFAULT 'i',
	fw_state_message   VARCHAR(128) NOT NULL DEFAULT '',

	CONSTRAINT device_pk PRIMARY KEY (device_id)
);
CREATE INDEX IF NOT EXISTS device_fk1 ON device(collection_id);
-- Indexes for IMSI and IMEI. In theory you could have devices with duplicate
-- IMEI and IMSI (if you move a SIM card from one device to another) but not
-- both at the same time. We will be using IMSI to identify the modules but
-- the users will most likely be more comfortable using the IMEI to locate their
-- devices since it is (usually) the easiest to retrieve. The uBlox modules have
-- the IMEI printed on top while the SIM chips we are using for the breakouts...
-- not so much.
CREATE UNIQUE INDEX IF NOT EXISTS device_imei ON device(imei);
CREATE UNIQUE INDEX IF NOT EXISTS device_imsi ON device(imsi);

-- Outputs from collections (ie collections of devices)
CREATE TABLE IF NOT EXISTS output (
	output_id     BIGINT      NOT NULL,
	collection_id BIGINT      NOT NULL REFERENCES collection (collection_id),
	output_type   VARCHAR(40) NOT NULL,
	config        JSON        NULL,
	enabled       BOOL        NOT NULL DEFAULT true,
	tags          JSON        NULL,

	CONSTRAINT output_pk PRIMARY KEY (output_id)
);

CREATE INDEX IF NOT EXISTS output_fk1 ON output (collection_id);

-- Invites contains user id, team id and a code. The user ID is the one
-- that created the invite.
CREATE TABLE IF NOT EXISTS invite (
	team_id  BIGINT      NOT NULL REFERENCES team (team_id),
	user_id  BIGINT      NOT NULL REFERENCES hordeuser (user_id),
	code     VARCHAR(64) NOT NULL,
	created  DATETIME    NOT NULL,

	CONSTRAINT invite_pk PRIMARY KEY (code)
);

CREATE INDEX IF NOT EXISTS invite_fk1 ON invite (team_id);
CREATE INDEX IF NOT EXISTS invite_fk2 ON invite (user_id);

CREATE TABLE IF NOT EXISTS device_lookup (
	imsi     BIGINT      NOT NULL, -- IMSI for device
	msisdn   VARCHAR(20) NOT NULL, -- MSISDN including country code
	icc      VARCHAR(30) NOT NULL,
	simtype  VARCHAR(20) NOT NULL,

	CONSTRAINT device_lookup_pk PRIMARY KEY(imsi)
);

-- Note that there are no explicit reference to the device table here
-- Ideally this should include a cascading delete for the IMSI but we'll
-- leave that for later when we know how it all pans out. Some manual
-- cleanups are necessary later on if devices are removed with the same
-- allocation. A trigger that removes the allocation if the device
-- is removed or the IMSI is changed is one way to solve this.
CREATE INDEX IF NOT EXISTS device_lookup_msisdn ON device_lookup(msisdn);
`

// DBAPNSchema is a big string with the DDL for the APN tables
const DBAPNSchema = `

CREATE TABLE IF NOT EXISTS apn (
	apn_id INT NOT NULL,
	name VARCHAR(16) NOT NULL,

	CONSTRAINT apn_pk PRIMARY KEY (apn_id)
);

CREATE INDEX IF NOT EXISTS apn_name ON apn(name);

INSERT INTO apn (apn_id, name) SELECT 0, 'default'
	WHERE NOT EXISTS (SELECT apn_id FROM apn WHERE apn_id = 0);

CREATE TABLE IF NOT EXISTS nas (
	nas_id INT NOT NULL,
	apn_id INT NOT NULL REFERENCES apn (apn_id),
	identifier VARCHAR(20) NOT NULL,
	cidr VARCHAR(64) NOT NULL,

	CONSTRAINT nas_pk PRIMARY KEY (apn_id, nas_id)
);
CREATE INDEX IF NOT EXISTS nas_nas_id ON nas(nas_id);
CREATE INDEX IF NOT EXISTS nas_apn_id ON nas(apn_id);

INSERT INTO nas (nas_id, identifier, cidr, apn_id) SELECT 0, 'NAS0', '127.0.0.1/24', 0
    WHERE NOT EXISTS (select nas_id FROM nas WHERE nas_id = 0 and apn_id = 0);

--
-- Storing IP addresses as strings is suboptimal but we'll keep this for
-- debugging purposes a little while longer. In the future this should
-- be replaced by a bigint (to allow for ipv6 addresses... some time in the
-- future)
--
CREATE TABLE IF NOT EXISTS nasalloc (
	nas_id INT NOT NULL,
	apn_id INT NOT NULL,
	imsi BIGINT NOT NULL,
	imei BIGINT NULL,
	ip   VARCHAR(32) NOT NULL,
	created DATETIME NOT NULL,

	FOREIGN KEY (apn_id, nas_id) REFERENCES nas (apn_id, nas_id),

	CONSTRAINT nasalloc_pk PRIMARY KEY (imsi, apn_id, nas_id)
);

CREATE INDEX IF NOT EXISTS nasalloc_apnid ON nasalloc(apn_id);
CREATE INDEX IF NOT EXISTS nasalloc_nasid ON nasalloc(nas_id);
CREATE INDEX IF NOT EXISTS nasalloc_imsi ON nasalloc(imsi);
CREATE INDEX IF NOT EXISTS nasalloc_ip ON nasalloc(ip);
`

// DBDataStoreSchema is the data store schema
const DBDataStoreSchema = `
-- This is the main storage schema for data
-- TODO(stalehd): Use regular integer IDs for collection
-- and device.
-- Note that this is an interim schema; changes are very
-- likely.
CREATE TABLE IF NOT EXISTS magpie_data (
	collection_id VARCHAR(64) NOT NULL,
	device_id     VARCHAR(64) NOT NULL,
	created       BIGINT      NOT NULL,
	inserted      BIGINT      NOT NULL,
	metadata      BYTES       NULL,
	payload       BYTES       NOT NULL);

	CREATE INDEX IF NOT EXISTS data_collection_id ON magpie_data(collection_id);
	CREATE INDEX IF NOT EXISTS data_device_id ON magpie_data(device_id);
	CREATE INDEX IF NOT EXISTS data_created ON magpie_data(created);
`

func (s *Schema) removeComments(schema string) string {
	ret := ""
	lines := strings.Split(schema, "\n")
	for _, v := range lines {
		line := v
		pos := strings.Index(line, "--")
		if pos == 0 {
			continue
		}
		if pos > 0 {
			line = line[0:pos]
		}
		if len(strings.TrimSpace(line)) == 0 {
			continue
		}
		ret += line + "\n"
	}
	return ret
}

func (s *Schema) changeDataTypes(cmd string) string {
	cmd = strings.Replace(cmd, "BYTE ", "SMALLINT ", -1)
	cmd = strings.Replace(cmd, "BYTES ", "BYTEA ", -1)
	cmd = strings.Replace(cmd, "JSON", "JSONB", -1)
	cmd = strings.Replace(cmd, "DATETIME", "TIMESTAMP", -1)
	return cmd
}

// DDL dumps the DDL statements including comments
func (s *Schema) DDL() []string {
	ret := []string{
		"-- --------------------------------------------------------------",
		fmt.Sprintf("-- DDL for database driver %s", s.driver),
		"-- --------------------------------------------------------------",
	}
	for _, v := range strings.Split(s.statements, "\n") {
		ret = append(ret, s.changeDataTypes(v))
	}

	return ret
}

// Statements returns an array of DDL statements
func (s *Schema) Statements() []string {
	var ret []string

	commands := strings.Split(s.removeComments(s.statements), ";")
	for _, v := range commands {
		if len(strings.TrimSpace(v)) > 0 {

			ret = append(ret, s.changeDataTypes(strings.TrimSpace(v)))
		}
	}
	return ret
}

// Create creates the database schema
func (s *Schema) Create(db *sql.DB) error {
	for i, v := range s.Statements() {
		_, err := db.Exec(v)
		if err != nil {
			return fmt.Errorf("unable to execute command #%d %s: %v", i, v, err)
		}
	}
	return nil
}

// NewSchema creates a new schema
func NewSchema(driver string, statements ...string) *Schema {

	return &Schema{driver: driver, statements: strings.Join(statements, "\n")}
}
