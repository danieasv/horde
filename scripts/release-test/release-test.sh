#!/bin/bash
# Test on released version with docker-compose
set -o errexit
set -o pipefail

if [ "${VERSION}" == "" ]; then
    echo "VERSION must be set when running the script"
    exit 1
fi

. scripts/release-test/test-functions.sh

echo "Clean up test stack"
docker-compose -f docker/docker-compose.yml kill
echo "Prune old images"
docker system prune --force --filter label=Type=Horde

echo "Build docker images"

cd docker
. build-docker-images.sh
cd ..


echo "Launch the test stack"
docker-compose -f docker/docker-compose.yml up --no-start
# Start the database first. The image needs a few moments to initialize
# properly and docker-compose just launches every image in succession
docker-compose -f docker/docker-compose.yml start

echo "Wait for stack to get up"
sleep 5

#
# These are the endpoints exposed by the docker-compose stack.
#
API_ENDPOINT=localhost:8080
MANAGEMENT=localhost:1234
RADIUS_ENDPOINT=127.0.0.1:1812
COAP_ENDPOINT=127.0.0.1:5683
UDP_ENDPOINT=127.0.0.1:31415

# This digs through the docker configuration to figure out which IP the UDP and CoAP requests get
# Convoluted.
CLIENT_UDP_IP=$(docker inspect docker_udp_1 | jq -r .[0].NetworkSettings.Networks.docker_default.Gateway)
CLIENT_COAP_IP=$(docker inspect docker_coap_1 | jq -r .[0].NetworkSettings.Networks.docker_default.Gateway)

RECEIVER_ENDPOINT=http://127.0.0.1:8282
RECEIVER_IP=$(docker inspect docker_messagereceiver_1 | jq -r .[0].NetworkSettings.Networks.docker_default.IPAddress)

test_version $VERSION
begin_run

. scripts/release-test/radius-defaults.sh
. scripts/release-test/radius.sh
. scripts/release-test/restapi_misc.sh
. scripts/release-test/restapi_team.sh
. scripts/release-test/restapi_collection.sh
. scripts/release-test/restapi_device.sh
. scripts/release-test/restapi_outputs.sh
. scripts/release-test/udp_coap_input.sh
. scripts/release-test/fota.sh

end_run

echo "Clean up environment"

echo "Kill the test stack"
docker-compose -f docker/docker-compose.yml kill


echo "----------------------------------------------------------------"
echo "          I'm making a note here: HUGE SUCCESS."
echo "                                           --GladOS"
