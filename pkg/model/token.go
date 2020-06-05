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
)

// Token is access tokens for the REST API
type Token struct {
	UserID   UserKey
	Resource string
	Write    bool
	Token    string
	Tags
}

// GenerateToken generates a new token. Note that this will overwrite the
// existing token
func (t *Token) GenerateToken() error {
	buf := make([]byte, 32)
	n, err := rand.Read(buf)
	if err == nil && n != len(buf) {
		return fmt.Errorf("unable to generate token %d bytes long. Only got %d bytes", len(buf), n)
	}
	t.Token = hex.EncodeToString(buf)
	return nil
}

// NewToken creates a new token
func NewToken() Token {
	return Token{Tags: NewTags()}
}
