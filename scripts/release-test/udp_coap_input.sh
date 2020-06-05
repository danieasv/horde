#!/bin/bash
#
# This script tests the CoAP and UDP interface. This assumes a running server on
# port 8080, ie the docker-compose stack. It expects to run at the root of the
# repository and that the docker-compose stack is running
#
set -o errexit
set -o pipefail


start_test "UDP and CoAP inputs, websocket outputs"
description "
Test the UDP and CoAP inputs by feeding a series of packets to the UDP and CoAP
endpoints while monitoring the output from the websocket on the collection. This
tests both internal routing and outputs."

section "Set up environment"

step "Create test user"
description "Create an API user"
bin/ctrlh user add --email=johndoe@example.com | grep Token | cut -f 3 -d ' ' > token.txt
TOKEN=$(cat token.txt)
rm token.txt

step "Create collection"
description "Create the collection for the test device"
curl -s --fail -HX-API-Token:${TOKEN} -XPOST -d'{"tags":{"name":"input-test-collection"}}' ${API_ENDPOINT}/collections | jq -r .collectionId > input-collection-id.txt
COLLECTION=$(cat input-collection-id.txt)
rm input-collection-id.txt

step "Create device"
description "Create the device that will be emulated by the test client."
curl -s --fail -HX-API-Token:${TOKEN} -XPOST -d'{"imsi":"2", "imei":"2"}' ${API_ENDPOINT}/collections/${COLLECTION}/devices | jq -r .deviceId > input-device-id.txt
DEVICE=$(cat input-device-id.txt)
rm input-device-id.txt

step "Create allocation for device"
description "Create a temporary allocation that points to the test client."
bin/ctrlh --endpoint=${MANAGEMENT} alloc add --apnid=0 --nasid=0 --imsi=2 --ip-address=${CLIENT_UDP_IP}


section "Test inputs"
step "Test CoAP and UDP input"
description "
A separate process will send UDP and CoAP messages to Horde, then attach to the
websocket and read the messages back.
"
echo "Running input test with token ${TOKEN} and command line bin/inputtest --coap-endpoint=${COAP_ENDPOINT} --collection-id=${COLLECTION} --token=${TOKEN} --udp-endpoint=${UDP_ENDPOINT} --repeat=10 "
bin/inputtest --coap-endpoint=${COAP_ENDPOINT} --collection-id=${COLLECTION} \
    --token=${TOKEN} --udp-endpoint=${UDP_ENDPOINT} --repeat=10

section "Clean up"
description "
Clean up the collections and devices that have been used during the test
"

step "Remove collection"
description "Remove collection from Horde"
curl -HX-API-Token:${TOKEN} -XDELETE ${API_ENDPOINT}/collections/${COLLECTION}/devices/${DEVICE}

step "Remove device"
description "Remove test device from Horde"
curl -HX-API-Token:${TOKEN} -XDELETE ${API_ENDPOINT}/collections/${COLLECTION}

step "Remove allocation"
description "Remove allocation from Horde"
bin/ctrlh --endpoint=${MANAGEMENT} alloc rm --apnid=0 --nasid=0 --imsi=2

end_test