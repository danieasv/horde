#!/bin/bash
#
# Test on master branch version running locally. Launch with "make run"
#
# Terminate on errors, print commands as they are executed.
set -o errexit
set -o pipefail
VERSION=develop

. scripts/release-test/test-functions.sh


#
# Local test
#
API_ENDPOINT=localhost:8080
MANAGEMENT=localhost:1234
COAP_ENDPOINT=127.0.0.1:5683
UDP_ENDPOINT=127.0.0.1:31415
CLIENT_UDP_IP=127.0.0.1
CLIENT_COAP_IP=127.0.0.1
RADIUS_ENDPOINT=127.0.0.1:1812
RECEIVER_ENDPOINT=http://127.0.0.1:8282
RECEIVER_IP=127.0.0.1

test_version $VERSION

# Used by the device downstream tests. This runs as a docker container in
# the full tests
bin/messagereceiver &

begin_run

. scripts/release-test/radius-defaults.sh
. scripts/release-test/restapi_misc.sh
. scripts/release-test/restapi_team.sh
. scripts/release-test/restapi_collection.sh
. scripts/release-test/restapi_device.sh
. scripts/release-test/restapi_outputs.sh
. scripts/release-test/udp_coap_input.sh
. scripts/release-test/fota.sh

end_run

echo Stop message receiver
curl -XPOST ${RECEIVER_ENDPOINT}/stop

echo "Great success, tests completed without errors"