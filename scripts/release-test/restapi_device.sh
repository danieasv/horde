#!/bin/bash

set -o errexit
set -o pipefail

start_test "REST API device resource test"


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

step "Create collection for device"
description "Create a collection to hold the devices."
COLLECTION=$(curl --fail -s -HX-API-Token:${TOKEN} -XPOST -d'{}' ${API_ENDPOINT}/collections | jq -r .collectionId)


section "Devices"
description "
Test the REST API for devices. Create, update, retrieve and delete devices via
the API"

step "Create a device"
description "
Create a new device with IMSI 10 and IMEI 20. The call should succeed.
"
DEVICE=$(curl -s --fail -HX-API-Token:${TOKEN} -d'{"imsi": "4711", "imei": "4711"}' ${API_ENDPOINT}/collections/${COLLECTION}/devices | jq -r .deviceId)

step "Create a device with non-numeric IMSI and IMEI"
description "
Create a device with non-numeric IMSI and IMEI strings. The call should fail
with a 400 Bad Request response.
"
TMP=$(curl -s -HX-API-Token:${TOKEN} -XPOST -d'{"imei": "thirty", "imsi": "one hundred million"}' ${API_ENDPOINT}/collections/${COLLECTION}/devices | jq -r .status)
if [ "$TMP" -ne "400" ]; then
    echo "Expected 400 response when fields are non-numeric"
    explode
fi

step "Create a device with no IMSI"
description "
Create a new device with just the IMEI field set. The call should fail with a
400 Bad Request response.
"
TMP=$(curl -s -HX-API-Token:${TOKEN} -XPOST -d'{"imsi": "30"}' ${API_ENDPOINT}/collections/${COLLECTION}/devices | jq -r .status)
if [ "$TMP" -ne "400" ]; then
    echo "Expected 400 response when no IMEI is set"
    explode
fi

step "Create a device with no IMEI"
description "
Create a new device with just the IMSI field set. The call should fail with a
400 Bad Request response.
"
TMP=$(curl -s -HX-API-Token:${TOKEN} -XPOST -d'{"imei": "30"}' ${API_ENDPOINT}/collections/${COLLECTION}/devices | jq -r .status)
if [ "$TMP" -ne "400" ]; then
    echo "Expected 400 response when no IMSI is set"
    explode
fi

step "Create a device with duplicate IMSI"
description "
IMSI for device must be unique. If a duplicate device is created the API must
respond with a 409 Conflict response.
"
TMP=$(curl -s -HX-API-Token:${TOKEN} -XPOST -d'{"imsi": "4711", "imei": "4712"}' ${API_ENDPOINT}/collections/${COLLECTION}/devices | jq -r .status)
if [ "$TMP" -ne "409" ]; then
    echo "Expected 400 response when there is a a duplicate IMSI"
    explode
fi

step "Create a device with duplicate IMEI"
description "
IMEIs must be unique. If a device with an existing IMEI is registered the API
must respond with a 409 Conflict response
"
TMP=$(curl -s -HX-API-Token:${TOKEN} -XPOST -d'{"imsi": "4712", "imei": "4711"}' ${API_ENDPOINT}/collections/${COLLECTION}/devices | jq -r .status)
if [ "$TMP" -ne "409" ]; then
    echo "Expected 400 response when there is a duplicate IMEI"
    explode
fi

step "Set tag on device"
TMP=$(curl -s -HX-API-Token:${TOKEN} -XPATCH -d'{"tags":{"some": "value"}}' ${API_ENDPOINT}/collections/${COLLECTION}/devices/${DEVICE} | jq -r .tags.some)
if [ "$TMP" != "value" ]; then
    echo "Expected field to be 'value' but it is $TMP"
    explode
fi

step "Query device"
description "
Query the device. The API should respond with the correct device.
"
curl -s --fail -HX-API-Token:${TOKEN} ${API_ENDPOINT}/collections/${COLLECTION}/devices/${DEVICE}

step "Remove tag from device"
curl --fail -s -HX-API-Token:${TOKEN} -XDELETE ${API_ENDPOINT}/collections/${COLLECTION}/devices/${DEVICE}/tags/some

step "Update tag on device"
description "
Update an existing tag with a new value. The updated value should be returned.
"
TMP=$(curl -s -HX-API-Token:${TOKEN} -XPATCH -d'{"tags":{"some": "other"}}' ${API_ENDPOINT}/collections/${COLLECTION}/devices/${DEVICE} | jq -r .tags.some)
if [ "$TMP" != "other" ]; then
    echo "Expected field to be 'other' but it is $TMP"
    explode
fi

step "Remove tag from device"
description "
Remove a tag from the device by sending a DELETE request to the API. The API
should return no error.
"
curl --fail -s -HX-API-Token:${TOKEN} -XDELETE ${API_ENDPOINT}/collections/${COLLECTION}/devices/${DEVICE}/tags/some


section "Downstream messages"
description "
Send messages downstream (from backend to device) via the API.
"

step "Message without payload"
description "
For obvious reasons the message parameter is required when sending a message.
The API should respond with a 400 bad request response if the message field is
not set.
"
TMP=$(curl -s -HX-API-Token:${TOKEN} -XPOST -d'{"transport":"udp", "port":4711}' ${API_ENDPOINT}/collections/${COLLECTION}/devices/${DEVICE}/to | jq -r .status)
if [ "$TMP" -ne "400" ]; then
    echo "Expected 400 response when message field is not set"
    explode
fi

step "UDP Message without port"
description "
For obvious reasons the port parameter is required when sending a message.
The API should respond with a 400 bad request response if the message field is
not set.
"
TMP=$(curl -s -HX-API-Token:${TOKEN} -XPOST -d'{"transport":"udp", "message":"Tm8gcG9ydCBmb3IgeW91Cg=="}' ${API_ENDPOINT}/collections/${COLLECTION}/devices/${DEVICE}/to | jq -r .status)
if [ "$TMP" -ne "400" ]; then
    echo "Expected 400 response when port field is not set"
    explode
fi

step "CoAP push without path"
description "
CoAP push messages needs the path parameter to work. If the path parameter is
missing from the request the server should respond with a 400 bad request
response.
"
TMP=$(curl -s -HX-API-Token:${TOKEN} -XPOST -d"{\"transport\":\"coap-push\", \"port\":4711, \"payload\":\"$message\"}" ${API_ENDPOINT}/collections/${COLLECTION}/devices/${DEVICE}/to | jq .status)
if [ "$TMP" -ne "400" ]; then
    echo "Expected downstream message to fail with a 400 response but it was $TMP"
    explode
fi

step "CoAP push without port"
description "
CoAP push messages needs the port parameter to work. If the port parameter is
missing from the request the server should respond with a 400 bad request
response.
"
TMP=$(curl -s -HX-API-Token:${TOKEN} -XPOST -d"{\"transport\":\"coap-push\", \"coapPath\":\"/foof\", \"payload\":\"$message\"}" ${API_ENDPOINT}/collections/${COLLECTION}/devices/${DEVICE}/to | jq .status)
if [ "$TMP" -ne "400" ]; then
    echo "Expected downstream message to fail with a 400 response but it was $TMP"
    explode
fi


section "Offline devices"
description "
Test downstream messages to offline devices. The device have no allocation and
haven't been online at all. The send operations should (eventually) fail.
"

step "Send data to offline device - UDP"
description "
Send a downstream message to the device via the API. Since the device doesn't
have an IP address the request should fail with a 409 Conflict response.
"
message=$(echo Hello world | base64)
TMP=$(curl -s -HX-API-Token:${TOKEN} -XPOST -d"{\"transport\":\"udp\", \"port\":4711, \"payload\":\"$message\"}" ${API_ENDPOINT}/collections/${COLLECTION}/devices/${DEVICE}/to | jq -r .status)
if [ "$TMP" -ne "409" ]; then
    echo "Expected downstream message to fail with a 409 response but it was $TMP"
    echo Command is curl -s -HX-API-Token:${TOKEN} -XPOST -d'{"transport":"udp", "port":4711, "payload":"$message"}' ${API_ENDPOINT}/collections/${COLLECTION}/devices/${DEVICE}/to
    echo Message=$message
    explode
fi


step "Send data to offline device - CoAP push"
description "
Send a downstream message to the device via the API. Since the device doesn't
have an IP address the request should fail. The message is sent as a CoAP POST
request to the device. The device is offline so the request should fail.
"
TMP=$(curl -s -HX-API-Token:${TOKEN} -XPOST -d"{\"transport\":\"coap-push\", \"coapPath\":\"/push\", \"port\":4711, \"payload\":\"$message\"}" ${API_ENDPOINT}/collections/${COLLECTION}/devices/${DEVICE}/to | jq -r .status)
if [ "$TMP" != "409" ]; then
    echo Command is curl -s -HX-API-Token:${TOKEN} -XPOST -d'{"transport":"coap-push", "coapPath":"/push", "port":4711, "payload":"$message"}' ${API_ENDPOINT}/collections/${COLLECTION}/devices/${DEVICE}/to
    echo "Expected downstream message to fail with a 409 response but it was $TMP"
    explode
fi

step "Send data to offline device - CoAP pull"
description "
Send a CoAP pull downstream message to the device via the API. The device is
responsible for sending the message so it won't generate an error.
"
curl --fail -s -HX-API-Token:${TOKEN} -XPOST -d"{\"transport\":\"coap-pull\", \"payload\":\"$message\"}" ${API_ENDPOINT}/collections/${COLLECTION}/devices/${DEVICE}/to


section "Online devices"
description "
Send downstream data to devices that have been online at any point. There's no
device out there so no data will be sent.
"

step "Create allocation for device"
description "
Create allocation for device. This makes the backend server send the message to
the device."
bin/ctrlh --endpoint=${MANAGEMENT} alloc add --apnid=0 --nasid=0 --imsi=4711 --ip-address=${RECEIVER_IP}
echo "Device has IP ${RECEIVER_IP}"

step "Update device with IP address"
description "
Set the allocated IP address for the device. This is normally set when there's
a RADIUS request sent to the server but we'll cheat this time
"
bin/radiustest --accept-expected -attr-imsi=4711 --attr-nas-identifier=NAS0 \
    --radius-endpoint=${RADIUS_ENDPOINT} --shared-secret=radiussharedsecret \
    --expected-cidr=0.0.0.0/0

step "Send CoAP pull data"
description "
CoAP pull data requests will succeed"
message=$(echo Hello world 2 | base64)
curl -s --fail -HX-API-Token:${TOKEN} -XPOST -d"{\"transport\":\"coap-pull\", \"payload\":\"$message\"}" ${API_ENDPOINT}/collections/${COLLECTION}/devices/${DEVICE}/to

step "Send UDP data"
description "
UDP message is sent to the device.
"
curl -s --fail -HX-API-Token:${TOKEN} -XPOST -d"{\"transport\":\"udp\", \"port\":4711, \"payload\":\"$message\"}" ${API_ENDPOINT}/collections/${COLLECTION}/devices/${DEVICE}/to

step "Send CoAP push data"
description "
CoAP push data should be sent to the device.
"
curl -s --fail -HX-API-Token:${TOKEN} -XPOST -d"{\"transport\":\"coap-push\", \"coapPath\":\"/push\", \"port\":4712, \"payload\":\"$message\"}" ${API_ENDPOINT}/collections/${COLLECTION}/devices/${DEVICE}/to

# Wait for requests to complete. There are a few roundtrips that
TMP="0"
echo "Wait for messages to drain"
count=60
until [ "$(curl -s ${RECEIVER_ENDPOINT}/coap-pull)" == "2" ] && [ "$count" -gt "0" ]; do
    sleep 1
    let count--
done
if [ "$count" == "0" ]; then
    echo "Timed out"
    explode
fi

step "Verify CoAP pull data"
description "
Read the CoAP pull data from the background process. The file should contain all
of the messages received via CoAP pull, numbered sequentially. Both the message
sent when the device was offline and when the device was online should be in the
file."
TMP=$(curl --fail -s ${RECEIVER_ENDPOINT}/coap-pull)
if [ "$TMP" -ne "2" ]; then
    echo "Expected 2 coap-pull messages but got $TMP"
    explode
fi

step "Verify CoAP push data"
description "
Rrad the CoAP push data sent to the emulated device. The file should contain the
message pushed while the device was online.
"
TMP=$(curl --fail -s ${RECEIVER_ENDPOINT}/coap-push)
if [ "$TMP" -ne "1" ]; then
    echo "Expected 1 coap-push message but got $TMP"
    explode
fi


step "Verify UDP push data"
description "
Read the UDP data sent to the device. The file should contain the message sent
to the device while it was online.
"
TMP=$(curl --fail -s ${RECEIVER_ENDPOINT}/udp)
if [ "$TMP" -ne "1" ]; then
    echo "Expected 1 udp message but got $TMP"
    explode
fi

section "Upstream data"
description "
Inspect the upstream data collection on the device. The collection should
contain all of the messages received by the backend server, 2 coap-pull,
1 coap-push, 1 udp, 4 in total.
"
curl --fail -v -s -HX-API-Token:${TOKEN}  ${API_ENDPOINT}/collections/${COLLECTION}/devices/${DEVICE}/data > upstream.txt
TMP=$(cat upstream.txt | jq '.messages | length')
if [ "$TMP" -ne "4" ]; then
    echo "Expected at least 4 items in the input queue but got $TMP"
    incorrect_count
fi
rm upstream.txt

section "Clean up"
step "Remove allocation"
description "Remove the temporary allocation for the device."
bin/ctrlh --endpoint=${MANAGEMENT} alloc rm --apnid=0 --nasid=0 --imsi=4711

step "Remove device"
description "Remove the device via the API."
curl --fail -s -HX-API-Token:${TOKEN} -XDELETE  ${API_ENDPOINT}/collections/${COLLECTION}/devices/${DEVICE}

step "Remove the collection"
description "Remove the collection via the API."
curl --fail -s -HX-API-Token:${TOKEN} -XDELETE  ${API_ENDPOINT}/collections/${COLLECTION}
