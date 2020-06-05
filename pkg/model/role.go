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
import "strings"

// RoleID is a single role.
type RoleID uint8

var (
	// AdminRole is the administrator role, ie with full access
	AdminRole = RoleID(1)
	// MemberRole is the member role, ie a member without administrative privileges
	MemberRole = RoleID(0)
)

// NewRoleIDFromString converts a string into a role ID
func NewRoleIDFromString(s string) RoleID {
	if strings.ToLower(s) == "admin" {
		return AdminRole
	}
	return MemberRole
}

// String returns the string representation of the role
func (r RoleID) String() string {
	if r == AdminRole {
		return "Admin"
	}
	return "Member"
}
