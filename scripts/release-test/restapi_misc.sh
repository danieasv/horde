#!/bin/bash

set -o errexit
set -o pipefail

start_test "REST API token, profile and system resource test"


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


section "Token resource"
step "/tokens resource"
description "
The token must be created by users that are authenticated throught the Telenor
ID or GitHub authentication providers. Ensure that clients using an API token
doesn't have access to /token or /session resources.
"
CODE=$(curl -s -HX-API-Token:${TOKEN} ${API_ENDPOINT}/tokens | jq -r .status)
if [ "${CODE}" -ne "403" ]; then
    echo "Expected 403 response from /tokens but got ${CODE}"
    echo "Command is curl -s -HX-API-Token:${TOKEN} ${API_ENDPOINT}/tokens"
    explode_here
fi


section "Profile resource"
step "/profile"
description "
The /profile resource shows the user profile. Ensure the profile type is set
to 'internal'
"
TMP=$(curl -s -HX-API-Token:${TOKEN} ${API_ENDPOINT}/profile | jq -r .provider)
if [ "${TMP}" != "internal" ]; then
    echo "Expected 'internal' provider type"
    explode_here
fi


section "System information resource"
description "
The system information resource shows system and version information
"
step "View system information"
curl -s -HX-API-Token:${TOKEN} ${API_ENDPOINT}/system > sys.txt

if [ "$(cat sys.txt | jq -r .version)" == "" ]; then
    echo "Missing version from system information"
    explode_here
fi

if [ "$(cat sys.txt | jq -r .buildData)" == "" ]; then
    echo "Missing build data from system information"
    explode_here
fi

if [ "$(cat sys.txt | jq -r .releaseName)" == "" ]; then
    echo "Missing release name from system information"
    explode_here
fi

rm sys.txt
