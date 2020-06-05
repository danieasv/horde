#!/bin/bash

set -o errexit
set -o pipefail

start_test "REST API output resource test"
description "
Test the REST API resources that represents an output. The actual forwarding is
tested in the input and output test, not in this. This test verifies that the
output resources behaves correctly.
"

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

step "Create collection for output"
description "Create a collection to hold the outputs"
COLLECTION=$(curl --fail -s -HX-API-Token:${TOKEN} -XPOST -d'{}' ${API_ENDPOINT}/collections | jq -r .collectionId)

section "Output resource"
description "Test general operations on output resources."

step "Create output"
description "Create an output via a POST call on the outputs resource"

OUTPUT=$(curl --fail -s -HX-API-Token:${TOKEN} -XPOST -d'{"type":"udp", "config":{"host":"127.0.0.1", "port":4711}}' ${API_ENDPOINT}/collections/${COLLECTION}/outputs | jq -r .outputId)
step "Create output with no request body"
description "
Outputs without the required fields will return a 400 bad request response from
the API. Ensure an empty request body gives this response.
"
TMP=$(curl -s -HX-API-Token:${TOKEN} -XPOST ${API_ENDPOINT}/collections/${COLLECTION}/outputs | jq -r .status)
if [ "$TMP" != "400" ]; then
    echo "Empty request body should return 400 status code"
    bad_api
fi

step "Retrieve unknown output"
description "Unknown output IDs should return a 404 error to the client."
TMP=$(curl -s -HX-API-Token:${TOKEN} -XGET ${API_ENDPOINT}/collections/${COLLECTION}/outputs/0 | jq -r .status)
if [ "$TMP" != "404" ]; then
    echo "Unknown output should return 404 status code but got $TMP"
    bad_api
fi

step "Create output with no output type"
description "
Outputs without the required fields will return a 400 bad request response from
the API. Test without the output type.
"
TMP=$(curl -s -HX-API-Token:${TOKEN} -XPOST -d'{"config": {"host": "a", "port":12}}' ${API_ENDPOINT}/collections/${COLLECTION}/outputs | jq -r .status)
if [ "$TMP" != "400" ]; then
    echo "Output missing required fields should return 400"
    bad_api
fi

step "Create output with no configuration"
description "
Outputs without the required fields will return a 400 bad request response from
the API. Test without the output configuration.
"
TMP=$(curl -s -HX-API-Token:${TOKEN} -XPOST -d'{"type":"mqtt"}' ${API_ENDPOINT}/collections/${COLLECTION}/outputs | jq -r .status)
if [ "$TMP" != "400" ]; then
    echo "Output missing required fields should return 400"
    bad_api
fi


step "Set tag on output"
description "
Set a new tag on the output. The new tag should be returned together with the
other properties on the output.
"
TMP=$(curl -s --fail -HX-API-Token:${TOKEN} -XPATCH -d'{"tags":{"other":"property"}}' ${API_ENDPOINT}/collections/${COLLECTION}/outputs/${OUTPUT} | jq -r .tags.other)
if [ "$TMP" != "property" ]; then
    echo "Should get the new tag value but got $TMP"
    echo "Query was curl -s -HX-API-Token:${TOKEN} -XPATCH -d'{"tags":{"other":"property"}}' ${API_ENDPOINT}/collections/${COLLECTION}/outputs/${OUTPUT}"
    echo "...and was looking for .tags.other in JSON"
    bad_api
fi


step "Update tag on output"
description "Update an existing tag.  The updated tag value should be returned."
TMP=$(curl -s --fail -HX-API-Token:${TOKEN} -XPATCH -d'{"tags":{"name":"a new name"}}' ${API_ENDPOINT}/collections/${COLLECTION}/outputs/${OUTPUT} | jq -r .tags.name)
if [ "$TMP" != "a new name" ]; then
    echo "Should get the updated tag value but got $TMP"
    bad_api
fi

step "Delete tag on output"
description "Delete a tag on on an output. The tag should be removed."
OLDVAL=$(curl -s --fail -HX-API-Token:${TOKEN} -XDELETE ${API_ENDPOINT}/collections/${COLLECTION}/outputs/${OUTPUT}/tags/some | jq -r .value)
if [ "$OLDVAL" != "a new name" ]; then
    echo "Should get the old value. Did not expect $OLDVAL"
fi

TMP=$(curl -s -HX-API-Token:${TOKEN} ${API_ENDPOINT}/collections/${COLLECTION}/outputs/${OUTPUT}/tags/some | jq -r .value)
if [ "$TMP" != "" ]; then
    echo "Should get empty value in return when looking for a deleted tag but got $TMP"
    bad_api
fi

step "Delete nonexisting tag on output"
description "
Delete a tag that doesn't exist. Deleting the same tag twice should return 204
no content for all requests.
"
curl -s --fail -HX-API-Token:${TOKEN} -XDELETE ${API_ENDPOINT}/collections/${COLLECTION}/outputs/${OUTPUT}/tags/some
curl -s  --fail -HX-API-Token:${TOKEN} -XDELETE ${API_ENDPOINT}/collections/${COLLECTION}/outputs/${OUTPUT}/tags/some

step "Change output type without changing the config"
description "Change the output type. It should fail unless the config is updated at the same time."
TMP=$(curl -s -HX-API-Token:${TOKEN} -XPATCH -d'{"type":"mqtt"}' ${API_ENDPOINT}/collections/${COLLECTION}/outputs/${OUTPUT} | jq -r .status)
if [ "$TMP" != "400" ]; then
    echo "Should get 400 when changing type but not config but got $TMP"
    bad_api
fi

step "Change output type and configuration"
description "
If the output type is changed the configuration must match. Ensure that the
output can be updated correctly.
"
TMP=$(curl -s -HX-API-Token:${TOKEN} -XPATCH -d'{"type":"webhook", "config":{"url": "http://localhost:8090/hook"}}' ${API_ENDPOINT}/collections/${COLLECTION}/outputs/${OUTPUT} | jq -r .outputId)
if [ "$TMP" != "${OUTPUT}" ]; then
    echo "Should get the updated output in return but got $TMP"
    bad_api
fi

step "Disable output"
description "
Ensure the output can be disabled. When the output is disabled it won't forward
any data.
"
TMP=$(curl -s -HX-API-Token:${TOKEN} -XPATCH -d'{"enabled": false}}' ${API_ENDPOINT}/collections/${COLLECTION}/outputs/${OUTPUT} | jq -r .enabled)
if [ "$TMP" != "false" ]; then
    echo "The updated output should be disabled but got $TMP"
    bad_api
fi

step "Enable output"
TMP=$(curl -s -HX-API-Token:${TOKEN} -XPATCH -d'{"enabled": true}}' ${API_ENDPOINT}/collections/${COLLECTION}/outputs/${OUTPUT} | jq -r .enabled)
if [ "$TMP" != "true" ]; then
    echo "The updated output should be disabled but got $TMP"
    bad_api
fi

step "View logs on output"
description "
The output logs shows a diagnostic log for the output.
"
curl --fail -s -HX-API-Token:${TOKEN} ${API_ENDPOINT}/collections/${COLLECTION}/outputs/${OUTPUT}/logs > /dev/null


step "View output status"
description "
The output status shows the number of messages forwarded, received and retries
plus an error count. Ensure the fields are shown for the output.
"
curl --fail -s -HX-API-Token:${TOKEN} ${API_ENDPOINT}/collections/${COLLECTION}/outputs/${OUTPUT}/status > errors.txt
TMP=$(cat errors.txt | jq -r .errorCount)
TMP=$(cat errors.txt | jq -r .forwarded)
TMP=$(cat errors.txt | jq -r .received)
TMP=$(cat errors.txt | jq -r .retries)
rm errors.txt


section "MQTT Outputs"
step "MQTT output without clientId"
description "
The MQTT outputs require a clientId, endpoint and topic name in the config.
Ensure that a missing client ID fails.
"
TMP=$(curl -s -HX-API-Token:${TOKEN} -XPOST -d'{"type":"mqtt", "config": {"endpoint": "tcp://localhost:1883", "topicName": "something"}}' ${API_ENDPOINT}/collections/${COLLECTION}/outputs | jq .status)
if [ "$TMP" != "400" ]; then
    echo "Should get 400 when missing client ID but got $TMP"
    bad_api
fi

step "MQTT output without endpoint"
description "
The MQTT outputs require a clientId, endpoint and topic name in the config.
Ensure that a missing endpoint fails.
"
TMP=$(curl -s -HX-API-Token:${TOKEN} -XPOST -d'{"type":"mqtt", "config": {"clientId": "horde", "topicName": "something"}}' ${API_ENDPOINT}/collections/${COLLECTION}/outputs | jq .status)
if [ "$TMP" != "400" ]; then
    echo "Should get 400 when missing endpoint but got $TMP"
    bad_api
fi

section "MQTT Outputs"
step "MQTT output without topic"
description "
The MQTT outputs require a clientId, endpoint and topic name in the config.
Ensure that a missing topic fails.
"
TMP=$(curl -s -HX-API-Token:${TOKEN} -XPOST -d'{"type":"mqtt", "config": {"endpoint": "tcp://localhost:1883", "clientId": "something"}}' ${API_ENDPOINT}/collections/${COLLECTION}/outputs | jq .status)
if [ "$TMP" != "400" ]; then
    echo "Should get 400 when missing topic but got $TMP"
    bad_api
fi

step "MQTT output with invalid endpoint"
description "
The endpoint parameter must be formatted as tcp://host:port or ssl://host:port.
Ensure invalid endpoint strings are rejected.
"
TMP=$(curl -s -HX-API-Token:${TOKEN} -XPOST -d'{"type":"mqtt", "config": {"endpoint": "something://weird", "topicName": "thetopic", "clientId": "something"}}' ${API_ENDPOINT}/collections/${COLLECTION}/outputs | jq .status)
if [ "$TMP" != "400" ]; then
    echo "Should get 400 for illegal endpoint but got $TMP"
    bad_api
fi


step "Create MQTT output"
description "
Create a disabled MQTT output with valid configuration.
"
TMP=$(curl -s -HX-API-Token:${TOKEN} -XPOST -d'{"type":"mqtt", "enabled": false, "config": {"endpoint": "tcp://localhost:1883", "topicName": "thetopic", "clientId": "something"}}' ${API_ENDPOINT}/collections/${COLLECTION}/outputs | jq .enabled)
if [ "$TMP" != "false" ]; then
    echo "Should get false for the enabled flag but got $TMP"
    bad_api
fi


section "UDP outputs"

step "New UDP output without host parameter"
description "
UDP outputs requires both a host and port parameter. Ensure that a missing host
parameter returns an error.
"
TMP=$(curl -s -HX-API-Token:${TOKEN} -XPOST -d'{"type":"udp", "config": { "port": 4711}}' ${API_ENDPOINT}/collections/${COLLECTION}/outputs | jq .status)
if [ "$TMP" != "400" ]; then
    echo "Should get 400 when missing client ID but got $TMP"
    bad_api
fi

step "New UDP output without port parameter"
description "
UDP outputs requires both a host and port parameter. Ensure that a missing port
parameter returns an error.
"
TMP=$(curl -s -HX-API-Token:${TOKEN} -XPOST -d'{"type":"udp", "config": {"host": "127.0.0.1"}}' ${API_ENDPOINT}/collections/${COLLECTION}/outputs | jq .status)
if [ "$TMP" != "400" ]; then
    echo "Should get 400 when missing client ID but got $TMP"
    bad_api
fi

step "New disabled UDP output"
description "
Ensure a new (disabled) UDP output with valid configuration can be created.
"
TMP=$(curl -s -HX-API-Token:${TOKEN} -XPOST -d'{"type":"udp", "enabled": false, "config": {"host": "127.0.0.1", "port": 4711}}' ${API_ENDPOINT}/collections/${COLLECTION}/outputs | jq .enabled)
if [ "$TMP" != "false" ]; then
    echo "New output should be disabled but got $TMP"
    bad_api
fi


section "Webhook outputs"
step "Webhook output without URL parameter"
description "
The webhook outputs require an URL parameter. Ensure you can't create one
without it.
"
TMP=$(curl -s -HX-API-Token:${TOKEN} -XPOST -d'{"type":"webhook", "config": {}}' ${API_ENDPOINT}/collections/${COLLECTION}/outputs | jq .status)
if [ "$TMP" != "400" ]; then
    echo "Should get 400 when missing URL parameter but got $TMP"
    bad_api
fi


step "Webhook with invalid URL"
description "
Ensure only valid URL parameters are accepted.
"
TMP=$(curl -s -HX-API-Token:${TOKEN} -XPOST -d'{"type":"webhook", "config": {"url": "postgres://my-host/database"}}' ${API_ENDPOINT}/collections/${COLLECTION}/outputs | jq .status)
if [ "$TMP" != "400" ]; then
    echo "Should get 400 when using invalid URL but got $TMP"
    bad_api
fi

step "Create disabled webhook"
description "
Ensure a (disabled) webhook can be created with valid parameters.
"
TMP=$(curl -s -HX-API-Token:${TOKEN} -XPOST -d'{"type":"webhook", "enabled": false, "config": {"url": "http://127.0.0.1/hook"}}' ${API_ENDPOINT}/collections/${COLLECTION}/outputs | jq .enabled)
if [ "$TMP" != "false" ]; then
    echo "Should get false when creating a disabled output but got $TMP"
    bad_api
fi


section "IFTTT outputs"
description "
IFTTT (If This Then That) is a popular service for simple serverless automation.
Ensure that the resource works as expected.
"

step "IFTTT without event name field"
description "
The IFTTT output requires both an event name and a key parameter. Ensure that
a missing event name returns errors.
"
TMP=$(curl -s -HX-API-Token:${TOKEN} -XPOST -d'{"type":"ifttt", "config": { "key": "secret"}}' ${API_ENDPOINT}/collections/${COLLECTION}/outputs | jq .status)
if [ "$TMP" != "400" ]; then
    echo "Should get 400 response when event name is missing but got $TMP"
    bad_api
fi

step "IFTTT without key field"
description "
The IFTTT output requires both an event name and a key parameter. Ensure that
a missing key returns errors.
"
TMP=$(curl -s -HX-API-Token:${TOKEN} -XPOST -d'{"type":"ifttt", "config": {"eventName": "1234"}}' ${API_ENDPOINT}/collections/${COLLECTION}/outputs | jq .status)
if [ "$TMP" != "400" ]; then
    echo "Should get 400 response when key is missing but got $TMP"
    bad_api
fi

step "Create IFTTT output"
description "
Create a (disabled) IFTTT output with presumed valid configuration.
"
TMP=$(curl -s -HX-API-Token:${TOKEN} -XPOST -d'{"type":"ifttt", "enabled": false, "config": {"eventName": "1234", "key": "secret"}}' ${API_ENDPOINT}/collections/${COLLECTION}/outputs | jq .enabled)
if [ "$TMP" != "false" ]; then
    echo "Should get false when creating a disabled output but got $TMP"
    bad_api
fi
