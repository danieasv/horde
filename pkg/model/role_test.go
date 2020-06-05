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
import "testing"

func TestRoleConversion(t *testing.T) {
	if NewRoleIDFromString("admin") != AdminRole {
		t.Fatal("Not admin")
	}
	if NewRoleIDFromString("ADMIN") != AdminRole {
		t.Fatal("Not admin")
	}
	if NewRoleIDFromString("Member") != MemberRole {
		t.Fatal("Not member")
	}
	if NewRoleIDFromString("member") != MemberRole {
		t.Fatal("Not member")
	}
	if NewRoleIDFromString("foo") != MemberRole {
		t.Fatal("not the default role")
	}

	if NewRoleIDFromString(AdminRole.String()) != AdminRole {
		t.Fatal("Not admin")
	}
	if NewRoleIDFromString(MemberRole.String()) != MemberRole {
		t.Fatal("not member")
	}
}
