#!/bin/bash
#
# This script tests the RADIUS interface. This won't work for the local
# development environment since it requires a relaunch of the APN processes.
#
set -o errexit
set -o pipefail

start_test "RADIUS"
description "
Test the APN with a custom set-up containing two NAS ranges. Both are served by
the same APN process.
"
section "Initialise test environment"
description "
Set up a separate APN and NAS for the RADIUS test. The RADIUS service should
accept requests from valid clients with a shared secret that matches the
configured secret. Requests with unknown NAS identifiers are rejected.
"
step "Set up APN"
bin/ctrlh apn add --apnid=1 --name="radius.mda"

step "Set up NAS 1 and NAS 2"
description "
Set up two NAS definitions, one named TEST1 that will receive allocations in the
127.0.0.1/24 range and another one, TEST2 that will receive allocations in the
127.1.0.1/24 range.
"
CIDR1=127.0.0.1/24
CIDR2=127.1.0.1/24
bin/ctrlh nas add --apnid=1 --identifier=TEST1 --nasid=1 --cidr=${CIDR1}
bin/ctrlh nas add --apnid=1 --identifier=TEST2 --nasid=2 --cidr=${CIDR2}

step "Create test user"
description "
The RADIUS server will reject unknown devices (via the RADIUS protocol) so we
need a predefined device. The device must be created by an user.
"
bin/ctrlh user add --email=johndoe@example.com | grep Token | cut -f 3 -d ' ' > token.txt
TOKEN=$(cat token.txt)
rm token.txt

step "Reload config"
description "
The new APN and NAS configuration requires a restart of the core service.
Re-launch the service when the APN and NAS is configured.
"
bin/ctrlh apn reload

step "Create collection"
curl -s -HX-API-Token:${TOKEN} -XPOST -d'{"tags":{"name":"radius-test-collection"}}' ${API_ENDPOINT}/collections | jq -r .collectionId > radius-collection-id.txt
COLLECTION=$(cat radius-collection-id.txt)
rm radius-collection-id.txt

step "Create device"
curl -s -HX-API-Token:${TOKEN} -XPOST -d'{"imsi":"1", "imei":"1"}' ${API_ENDPOINT}/collections/${COLLECTION}/devices | jq -r .deviceId > radius-device-id.txt
DEVICE=$(cat radius-device-id.txt)
rm radius-device-id.txt

echo "------------ ===== APN List ===== --------------"
bin/ctrlh apn list
bin/ctrlh nas list --apnid=1
echo "------------ ===== APN List ===== --------------"

section "Test RADIUS server"

step "Regular radius request"
description "
Run a standard RADIUS request from the TEST1 NAS. The IMSI is set to the
device's IMSI.
"
bin/radiustest --accept-expected=true --attr-imsi 1 --attr-nas-identifier TEST1 --expected-cidr=${CIDR1} --radius-endpoint=${RADIUS_ENDPOINT} --shared-secret radiussharedsecret

step "Regular radius request to APN 2"
description "
Run a standard RADIUS request from the TEST2 NAS. The imsi is set to the
device's IMSI.
"
bin/radiustest --accept-expected=true --attr-imsi 1 --attr-nas-identifier TEST2 --expected-cidr=${CIDR2} --radius-endpoint=${RADIUS_ENDPOINT} --shared-secret radiussharedsecret

step "Request from unknown NAS"
description "
Use an unknown NAS identifier in the RADIUS request. The request should be
rejected by the server.
"
bin/radiustest --accept-expected=false --attr-imsi 1 --attr-nas-identifier TEST9 --radius-endpoint=${RADIUS_ENDPOINT} --shared-secret radiussharedsecret

step "Incorrect shared secret"
description "
Use an incorrect shared secret for the RADIUS request. The request will be a
valid RADIUS request but the fields are encrypted with the shared secret so the
server should not respond to the request even if the parameters are valid.
"
bin/radiustest --failure=true --attr-imsi 1 --attr-nas-identifier TEST1 --radius-endpoint=${RADIUS_ENDPOINT} --shared-secret radiussharedsecretx

step "Invalid IMSI"
description "
Use an unknown IMSI in the request. The RADIUS server should reject the request.
"
bin/radiustest --accept-expected=false --attr-imsi 2 --attr-nas-identifier TEST1 --radius-endpoint=${RADIUS_ENDPOINT} --shared-secret radiussharedsecret


section "Clean up"

step "Remove allocations"
bin/ctrlh  alloc rm --apnid=1 --nasid=1 --imsi=1
bin/ctrlh  alloc rm --apnid=1 --nasid=2 --imsi=1

step "Remove NAS 1 and NAS 2"
bin/ctrlh nas rm --apnid=1 --nasid=2
bin/ctrlh nas rm --apnid=1 --nasid=1

step "Remove APN"
bin/ctrlh apn rm --apnid=1

step "Remove device"
curl -HX-API-Token:${TOKEN} -XDELETE ${API_ENDPOINT}/collections/${COLLECTION}/devices/${DEVICE}

step "Remove collection"
curl -HX-API-Token:${TOKEN} -XDELETE ${API_ENDPOINT}/collections/${COLLECTION}

step "Reload config"
description "
The new APN and NAS configuration requires a restart of the core service.
Re-launch the service when the APN and NAS is configured.
"
bin/ctrlh apn reload

end_test
