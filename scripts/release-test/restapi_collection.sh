#!/bin/bash

set -o errexit
set -o pipefail

start_test "REST API collection resource test"

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

section "Collections"
description "Test CRUD operations on collections via the REST API."

step "Create a collection"
description "Create a new collection via the API. The 'name' tag is set on the collection"
COLLECTION=$(curl -s --fail -HX-API-Token:${TOKEN} -XPOST -d'{"tags": {"name": "The test collection"}}' -XPOST ${API_ENDPOINT}/collections | jq -r .collectionId)

step "Set all field masks to false"
description "Set all of the field masks to the default (false)"
curl -s --fail -HX-API-Token:${TOKEN} -XPATCH -d'{"fieldMask": {"imsi": false, "imei": false, "msisdn": false, "location": false}}' ${API_ENDPOINT}/collections/${COLLECTION} > /dev/null

step "Update IMSI field mask on collection"
description "
Set the IMSI field mask on the collection. The default value is 'false', ie no
filtering. Turn on IMSI filtering and the field mask field should return true
"
TMP=$(curl -s --fail -HX-API-Token:${TOKEN} -XPATCH -d'{"fieldMask": {"imsi": true}}' ${API_ENDPOINT}/collections/${COLLECTION} | jq -r .fieldMask.imsi)
if [ "$TMP" != "true" ]; then
    echo "Expected field mask on IMSI to be set but it was $TMP"
    controlled_demolition
fi

step "Update IMEI field mask on collection"
description "
Set the IMEI field mask on the collection. Ensure it is correctly set.
"
TMP=$(curl -s --fail -HX-API-Token:${TOKEN} -XPATCH -d'{"fieldMask": {"imei": true}}' ${API_ENDPOINT}/collections/${COLLECTION} | jq -r .fieldMask.imei)
if [ "$TMP" != "true" ]; then
    echo "Expected field mask on IMEI to be set but it was $TMP"
    controlled_demolition
fi

# MSISDN field mask is ignored since it isn't used

step "Update location field mask on collection"
description "
Set the location field mask on the collection. Ensure it is set.
"
TMP=$(curl -s --fail -HX-API-Token:${TOKEN} -XPATCH -d'{"fieldMask": {"location": true}}' ${API_ENDPOINT}/collections/${COLLECTION} | jq -r .fieldMask.location)
if [ "$TMP" != "true" ]; then
    echo "Expected field mask on location to be set but it was $TMP"
    controlled_demolition
fi

step "Set tag on collection"
description "
Create a new tag on the collection. Ensure the value has been set correctly.
"
TMP=$(curl -s --fail -HX-API-Token:${TOKEN} -XPATCH -d'{"tags": {"some": "value"}}' ${API_ENDPOINT}/collections/${COLLECTION} | jq -r .tags.some)
if [ "$TMP" != "value" ]; then
    echo "Expected tag to be set but it was $TMP"
    implode_in_inventive_ways
fi

step "Delete tag in collection"
description "Delete a tag by sending a DELETE request to the tag directly."
curl -s --fail -HX-API-Token:${TOKEN} -XDELETE ${API_ENDPOINT}/collections/${COLLECTION}/tags/some

step "Delete collection"
description "Remove the collection by sending a DELETE request"
curl -s --fail -HX-API-Token:${TOKEN} -XDELETE ${API_ENDPOINT}/collections/${COLLECTION}

