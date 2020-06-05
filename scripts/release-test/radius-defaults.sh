#!/bin/bash
#
# This script tests the RADIUS interface with the default configuration.
#
set -o errexit
set -o pipefail

# this is the default CIDR
CIDR=127.0.0.1/24

start_test "RADIUS - default config"
description "
Test the RADIUS default configuration. This uses the default built-in APN and
NAS configuration.
"
section "Initialise test environment"
step "Create test user"
description "
The RADIUS server will reject unknown devices (via the RADIUS protocol) so we
need a predefined device. The device must be created by an user.
"
bin/ctrlh user add --email=johndoe@example.com | grep Token | cut -f 3 -d ' ' > token.txt
TOKEN=$(cat token.txt)
rm token.txt

step "Create collection"
description "Create a test collection via the API"
curl -s -HX-API-Token:${TOKEN} -XPOST -d'{"tags":{"name":"radius-test-collection"}}' ${API_ENDPOINT}/collections | jq -r .collectionId > radius-collection-id.txt
COLLECTION=$(cat radius-collection-id.txt)
rm radius-collection-id.txt

step "Create device"
description "Create a test device via the API"
curl -s -HX-API-Token:${TOKEN} -XPOST -d'{"imsi":"1", "imei":"1"}' ${API_ENDPOINT}/collections/${COLLECTION}/devices | jq -r .deviceId > radius-device-id.txt
DEVICE=$(cat radius-device-id.txt)
rm radius-device-id.txt

section "Test RADIUS server"

step "Regular radius request to APN 0"
description "
Run a standard RADIUS request from the NAS0 NAS. The IMSI is set to the
device's IMSI.
"
bin/radiustest --accept-expected=true --attr-imsi 1 --attr-nas-identifier NAS0 --expected-cidr=${CIDR} --radius-endpoint=${RADIUS_ENDPOINT} --shared-secret radiussharedsecret

step "Request from unknown NAS"
description "
Use an unknown NAS identifier in the RADIUS request. The request should be
rejected by the server.
"
bin/radiustest --accept-expected=false --attr-imsi 1 --attr-nas-identifier NAS99 --radius-endpoint=${RADIUS_ENDPOINT} --shared-secret radiussharedsecret

step "Incorrect shared secret"
description "
Use an incorrect shared secret for the RADIUS request. The request will be a
valid RADIUS request but the fields are encrypted with the shared secret so the
server should not respond to the request even if the parameters are valid.
"
bin/radiustest --failure=true --attr-imsi 1 --attr-nas-identifier NAS0 --radius-endpoint=${RADIUS_ENDPOINT} --shared-secret radiussharedsecretx

step "Invalid IMSI"
description "
Use an unknown IMSI in the request. The RADIUS server should reject the request.
"
bin/radiustest --accept-expected=false --attr-imsi 2 --attr-nas-identifier NAS0 --radius-endpoint=${RADIUS_ENDPOINT} --shared-secret radiussharedsecret


section "Clean up"

step "Remove allocation"
description "Remove the IP allocation used to test the RADIUS server"
bin/ctrlh alloc rm --apnid=0 --nasid=0 --imsi=1

step "Remove device"
description "Delete the test device via the API"
curl -HX-API-Token:${TOKEN} -XDELETE ${API_ENDPOINT}/collections/${COLLECTION}/devices/${DEVICE}

step "Remove collection"
description "Delete the test collection via the API"
curl -HX-API-Token:${TOKEN} -XDELETE ${API_ENDPOINT}/collections/${COLLECTION}


end_test
