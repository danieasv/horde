package model
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
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// Invite is a code, a creation date and an inviter (aka UserID)
type Invite struct {
	Code    string
	UserID  UserKey
	TeamID  TeamKey
	Created time.Time
}

func createCode() (string, error) {
	buf := make([]byte, 16)
	n, err := rand.Read(buf[0:15][:])
	if n != len(buf)-1 {
		return "", fmt.Errorf("unable to read random bytes. Needed %d bytes, got %d", len(buf)-1, n)
	}
	if err != nil {
		return "", err
	}
	// Add a checksum at the end. It won't tell much but it will catch any
	// typos at the end
	chk := byte(42)
	for i := range buf {
		chk ^= buf[i]
	}
	buf[15] = chk
	return hex.EncodeToString(buf), nil
}

// NewInvite creates a new invite
func NewInvite(user UserKey, team TeamKey) (Invite, error) {
	code, err := createCode()
	return Invite{Code: code, UserID: user, TeamID: team, Created: time.Now()}, err
}

// ValidInviteCode checks if the code is valid
func ValidInviteCode(code string) bool {
	buf, err := hex.DecodeString(code)
	if err != nil {
		return false
	}
	if len(buf) != 16 {
		return false
	}
	chk := byte(42)
	for i := 0; i < 15; i++ {
		chk ^= buf[i]
	}
	return chk == buf[15]
}
