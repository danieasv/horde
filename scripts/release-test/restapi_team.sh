#!/bin/bash

set -o errexit
set -o pipefail

start_test "REST API token, profile and team resource test"

section "Set up environment"
step "Create access token"
description "
Create a user that will be used to access the REST API. The token has full
access to all resources in the API with a few exceptions.
"
bin/ctrlh user add --email=apiuser@example.com > tmp.txt
TOKEN=$(cat tmp.txt | grep Token | cut -f 3 -d ' ')
USERID=$(cat tmp.txt | grep ID | cut -f 3 -d ' ')
rm tmp.txt

step "Create 2nd user"
description "
The second user will be used to verify that invites works
"
bin/ctrlh user add --email=apiuser2@example.com > tmp.txt
TOKEN2=$(cat tmp.txt | grep Token | cut -f 3 -d ' ')
USERID2=$(cat tmp.txt | grep ID | cut -f 3 -d ' ')
rm tmp.txt

section "Team resource"
step "Create a team"
TEAM=$(curl -s -HX-API-Token:${TOKEN} -XPOST -d'{"tags":{"name": "API token team"}}' ${API_ENDPOINT}/teams | jq -r .teamId)

step "Retrieve tag on team"
TMP=$(curl -s -HX-API-Token:${TOKEN} ${API_ENDPOINT}/teams/${TEAM}/tags/name | jq -r .value)
if [ "${TMP}" != "API token team" ]; then
    echo "Expected tag value for team ${TEAM}  but got ${TMP}"
    echo "Command was curl -s -HX-API-Token:${TOKEN} ${API_ENDPOINT}/teams/${TEAM}/tags/name | jq -r .value"
    explode_here
fi

step "Update the team"
curl -s -HX-API-Token:${TOKEN} -XPATCH -d'{"tags": {"name": "A new name"}}' ${API_ENDPOINT}/teams/${TEAM} > /dev/null

step "Retrieve updated tag on team"
TMP=$(curl -s -HX-API-Token:${TOKEN} ${API_ENDPOINT}/teams/${TEAM}/tags/name | jq -r .value)
if [ "${TMP}" != "A new name" ]; then
    echo "Expected tag value but got ${TMP}"
    echo "Command was curl -s -HX-API-Token:${TOKEN} ${API_ENDPOINT}/teams/${TEAM}/tags/name | jq -r .value"
    explode_here
fi

step "Delete tag from team"
curl -s -HX-API-Token:${TOKEN} -XDELETE ${API_ENDPOINT}/teams/${TEAM}/tags/name


section "Invites"
description "
Invites are generate by one user and accepted by another. There is no API call
to add an user to a team so all team joins is done through invites. This means
that we don't have to handle privacy issues -- if you accept an invite you are
implicitly giving a consent to share your name with the other team members.

There is no team with other teams as members, ie everyone is a direct member of
a team.
"
step "Create invite to team"
CODE=$(curl -s -HX-API-Token:${TOKEN} -XPOST ${API_ENDPOINT}/teams/${TEAM}/invites | jq -r .code)

step "Remove invite"
TMP=$(curl -s -HX-API-Token:${TOKEN} -XPOST ${API_ENDPOINT}/teams/${TEAM}/invites | jq -r .code)
curl -s  -HX-API-Token:${TOKEN} -XDELETE ${API_ENDPOINT}/teams/${TEAM}/invites/${TMP}

step "Accept invite"
description "
Use the 2nd user to accept the invite to the team. This will add the user as a
member of the team the first user has created.
"
JOINED_TEAM=$(curl -s -HX-API-Token:${TOKEN2} -XPOST -d"{\"code\":\"${CODE}\"}" ${API_ENDPOINT}/teams/accept | jq -r .teamId)
if [ "${JOINED_TEAM}" != "${TEAM}"]; then
    echo "Did not join the same team"
    echo "Command was curl -s -HX-API-Token:${TOKEN2} -XPOST -d"{\"code\":\"${CODE}\"}" ${API_ENDPOINT}/teams/accept | jq -r .teamId"
    explode_here
fi

step "Ensure accept codes are valid only once"
description "
Invite codes can only be used once, then they are deleted. Using the same code a
second time should return a 404 status code.
"
TMP=$(curl -s -HX-API-Token:${TOKEN2} -XPOST -d"{\"code\":\"${CODE}\"}" ${API_ENDPOINT}/teams/accept | jq -r .status)
if [ "${TMP}" -ne "404" ]; then
    echo "Expected a 404 return on a previously used invite code"
    echo "Command was curl -s -HX-API-Token:${TOKEN2} -XPOST -d"{\"code\":\"${CODE}\"}" ${API_ENDPOINT}/teams/accept | jq -r .status"
    explode_here
fi

step "List members of team"
description "
Both users should be able to retrieve the user information on the other user via
direct GET calls.
"
OTHER2=$(curl -s -HX-API-Token:${TOKEN} ${API_ENDPOINT}/teams/${TEAM}/members/${USERID2} | jq -r .userId)
OTHER1=$(curl -s -HX-API-Token:${TOKEN2} ${API_ENDPOINT}/teams/${JOINED_TEAM}/members/${USERID} | jq -r .userId)
if [ "${OTHER2}" != "${USERID2}" ]; then
    echo "User 1 should see user 1 in the team list"
    echo "Command was curl -s -HX-API-Token:${TOKEN} ${API_ENDPOINT}/teams/${TEAM}/members/${USERID2} | jq -r .userId"
    echo "Command was curl -s -HX-API-Token:${TOKEN2} ${API_ENDPOINT}/teams/${JOINED_TEAM}/members/${USERID} | jq -r .userId"
    explode_here
fi
if [ "${OTHER1}" != "${USERID}" ]; then
    echo "User 2 should see user 1 in the team list"
    echo "Command was curl -s -HX-API-Token:${TOKEN} ${API_ENDPOINT}/teams/${TEAM}/members/${USERID2} | jq -r .userId"
    echo "Command was curl -s -HX-API-Token:${TOKEN2} ${API_ENDPOINT}/teams/${JOINED_TEAM}/members/${USERID} | jq -r .userId"
    explode_here
fi

step "Set member to admin"
description "
The first user should now be able to set the other user as an admin for the team
"
TMP=$(curl -s -HX-API-Token:${TOKEN} -XPATCH -d'{"role":"admin"}'  ${API_ENDPOINT}/teams/${TEAM}/members/${USERID2} | jq -r .role)
if [ "${TMP}" != "Admin" ]; then
    echo "Other user should now be admin of team"
    explode_here
fi

step "Non-creator updates team"
description "
Now that the 2nd user is an admin he/she is able to update the team. Add a tag
to verify.
"
TMP=$(curl -s -HX-API-Token:${TOKEN2} -XPATCH -d'{"tags":{"mode":"I want to own this team"}}' ${API_ENDPOINT}/teams/${TEAM} | jq -r .tags.mode)
if [ "${TMP}" != "I want to own this team" ]; then
    echo "Could not update team tag"
    explode_here
fi

step "Set member from admin to regular member"
description "
The 2nd user sets the original creator of the team as a standard read-only
member.
"
TMP=$(curl -s -HX-API-Token:${TOKEN2} -XPATCH -d'{"role":"member"}'  ${API_ENDPOINT}/teams/${TEAM}/members/${USERID} | jq -r .role)
if [ "${TMP}" != "Member" ]; then
    echo "Original creator should now be member of team"
    explode_here
fi

step "Ensure member can't update team"
description "
The original creator is now a regular member of the team and aren't allowed to
update the team anymore.
"
TMP=$(curl -s -HX-API-Token:${TOKEN} -XPATCH -d'{"tags":{"mode":"But this is *my* team"}}' ${API_ENDPOINT}/teams/${TEAM} | jq -r .status)
if [ "${TMP}" -ne "403" ]; then
    echo "Should not be allowed to update the team. Expected 403 but got ${TMP}"
    echo Command is curl -s -HX-API-Token:${TOKEN} -XPATCH -d'{"tags":{"mode":"But this is *my* team"}}' ${API_ENDPOINT}/teams/${TEAM}
    explode_here
fi

step "Remove member from team"
curl -HX-API-Token:${TOKEN2} -XDELETE  ${API_ENDPOINT}/teams/${TEAM}/members/${USERID}

step "Original team owner can't find the team"
TMP=$(curl -s -HX-API-Token:${TOKEN} ${API_ENDPOINT}/teams/${TEAM} |jq -r .status)
if [ "${TMP}" -ne "404" ]; then
    echo "Original team owner should not be able to query team"
    explode_here
fi
