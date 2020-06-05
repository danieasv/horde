#!/bin/bash -x
#
# This script tests the firmware-over-the-air functions. This assumes a running
# server on port 8080, ie the docker-compose stack. It expects to run at the
# root of the repository and that the docker-compose stack is running
#
set -o errexit
set -o pipefail
# Enable this for debugging
# set -o xtrace
# Check device state for the device
function check_state {
    target=$1
    sleep 1
    curl -s --fail -HX-API-Token:${TOKEN} ${API_ENDPOINT}/collections/${COLLECTION}/devices/${DEVICE} | jq -r .firmware.state > state.txt
    STATE=$(cat state.txt)
    rm state.txt
    if [ "$target" != "$STATE" ]; then
        echo "Firmware state is not $target (state is $STATE)"
        explode_here
    fi
}

start_test "FOTA"
description "
Firmware-over-the-air test that uses test tools to emulate a device that does
a FOTA process. There are three kinds of FOTA processes tested; the simplest
which just will download the image from the firmware resource, a simplified that
will post the device metadata to a known resource and then receive a response
with a pointer to the resource with the updated firmware and a LwM2M-backed
process that uses the LwM2M protocol on top of CoAP to control the firmware
update.
"
section "Initialise test environment"

# Create a test user. This uses the default core endpoint
step "Create test user"
description "
Create the user that will own the collection, device and firmware images.
"
bin/ctrlh user add --email=johndoe@example.com | grep Token | cut -f 3 -d ' ' > token.txt
TOKEN=$(cat token.txt)
rm token.txt

step "Create collection"
description "
Create the collection that will include the firmware images. The collection is
created with firmware management set to device.
"
curl -s --fail -HX-API-Token:${TOKEN} -XPOST -d'{"firmware":{"management": "device"}, "tags":{"name":"fota-test-collection"}}' ${API_ENDPOINT}/collections | jq -r .collectionId > fota-collection-id.txt
COLLECTION=$(cat fota-collection-id.txt)
rm fota-collection-id.txt

step "Create device"
description "
Create the device that will receive the firmware upgrades.
"
curl -s --fail -HX-API-Token:${TOKEN} -XPOST -d'{"imsi":"3", "imei":"3"}' ${API_ENDPOINT}/collections/${COLLECTION}/devices | jq -r .deviceId > fota-device-id.txt
DEVICE=$(cat fota-device-id.txt)
rm fota-device-id.txt

step "Allocate IP for device"
description "
Pre-allocate the device's IP address. This allows us to skip the RADIUS step
which will happen automatically during connect. Since we don't have a device
during the test but use verification tools we preallocate the device.
"
bin/ctrlh --endpoint=${MANAGEMENT} alloc add --apnid=0 --nasid=0 --imsi=3 --ip-address=${CLIENT_COAP_IP}


section "No firmware uploaded to collection"
description "
No firmware image is uploaded to the collection. The device should not get a new
image when it reports back to the Horde service.
"

step "Direct download from CoAP resource, with no new firmware"
bin/fotaclient --direct --horde-endpoint=${COAP_ENDPOINT} --no-new --version=5.2.1

step "Simplified FOTA with no new firmware"
bin/fotaclient --simple --horde-endpoint=${COAP_ENDPOINT} --no-new --version=5.1.2

step "LwM2M with no firmware uploaded"
bin/fotaclient --horde-endpoint=${COAP_ENDPOINT} --scenario=noupdate --version=4.1.9


# Create dummy firmware files, one 4096 bytes, one 5150
section "Upload firmware"
description "
Upload two firmware images to the server. One is 4096 bytes and will fit exactly
into N blocks of a block-wise transfer in CoAP while the other is 5150 bytes and
will have an additional block at the end of the transfer.
"
head -c 128 /dev/urandom > firmware1.bin
head -c 128 /dev/urandom > firmware2.bin

step "Upload image 1 (4096 bytes)"
curl -s --fail -HX-API-Token:${TOKEN} -XPOST -F image=@firmware1.bin ${API_ENDPOINT}/collections/${COLLECTION}/firmware | jq -r .imageId > image1.txt
IMAGE1=$(cat image1.txt)
curl -s --fail -HX-API-Token:${TOKEN} -XPATCH -d'{"version": "1.0.0"}' ${API_ENDPOINT}/collections/${COLLECTION}/firmware/${IMAGE1} > /dev/null
rm firmware1.bin
rm image1.txt

step "Upload image 2 (5150 bytes)"
curl -s --fail  -HX-API-Token:${TOKEN} -XPOST -F image=@firmware2.bin ${API_ENDPOINT}/collections/${COLLECTION}/firmware | jq -r .imageId > image2.txt
IMAGE2=$(cat image2.txt)
curl -s --fail -HX-API-Token:${TOKEN} -XPATCH -d'{"version": "2.0.0"}' ${API_ENDPOINT}/collections/${COLLECTION}/firmware/${IMAGE2} > /dev/null
rm firmware2.bin
rm image2.txt


section "Simple FOTA"
description "
Do a simple FOTA process.
"
step "Set target to 1.0.0 for device 1"
curl -s --fail -HX-API-Token:${TOKEN} -XPATCH -d"{\"firmware\":{\"targetFirmwareId\": \"${IMAGE1}\"}}" ${API_ENDPOINT}/collections/${COLLECTION}/devices/${DEVICE} > /dev/null

step "Device is up to date (version 1.0.0)"
description "
Set the target to 1.0.0 (the default) for the device. If a device reports back
with this version number it will be flagged as up-to-date and there will be no
further actions taken.
"
bin/fotaclient --simple --horde-endpoint=${COAP_ENDPOINT} --no-new --version=1.0.0

step "Device is out of date (0.9.0 -> 1.0.0)"
description "
When the device reports version 0.9.0 it will trigger the upgrade process.
"
bin/fotaclient --simple --horde-endpoint=${COAP_ENDPOINT} --version=0.9.0


section "L2M2M FOTA"
step "Test with disabled firmware management (on device)"
description "
Firmware management is disabled (the default for new collections) so the device
won't get an upgrade request when it reports back. The check is independent for
both the simple and LwM2M implementations so there's no need to test both
scenarios. Since the L2M2M upgrade process is a lot more involved we'll test the
different alternatives here.
"

curl -s --fail -HX-API-Token:${TOKEN} -XPATCH -d'{"firmware":{"management":"disabled"}}' ${API_ENDPOINT}/collections/${COLLECTION} > /dev/null
bin/fotaclient --horde-endpoint=${COAP_ENDPOINT} --scenario=noupdate --version=1.0.0

step "Enable collection-managed firmware"
description "
Turn on collection management for the firmware. The target firmware is set on
the collection the device belongs to and all the devices in the collection will
receive an upgrade.
"
curl -s --fail -HX-API-Token:${TOKEN} -XPATCH -d"{\"firmware\":{\"management\":\"collection\", \"targetFirmwareId\":\"${IMAGE1}\"}}" ${API_ENDPOINT}/collections/${COLLECTION} > /dev/null

step "Test with enabled firmware management (on collection), current"
description "
Device reports back with a current version. No further action is taken.
"
bin/fotaclient --horde-endpoint=${COAP_ENDPOINT} --scenario=noupdate --version=1.0.0

step "Ensure the state is set to current on an up-to-date device"
description "
Devices that have up-to-date firmware versions should have the state set to
Current when they report back.
"
check_state "Current"

step "Test with enabled firmware management (on collection), update"
description "
Device reports back with an outdated version. Upgrade is triggered.
"
bin/fotaclient --horde-endpoint=${COAP_ENDPOINT} --version=0.9.0 --scenario=update

step "Verify version is set to 'Completed'"
description "
Verify that the device state is set to Completed, ie completed download. The
state isn't changed to current until it has checked in with the updated version.
"
check_state "Completed"

step "Call back with 1.0.0 and ensure state is current"
bin/fotaclient --horde-endpoint=${COAP_ENDPOINT} --scenario=noupdate --version=1.0.0
check_state "Current"

step "Enable device-level management"
description "
Turn on device-level management. Individual devices in the collection is managed
one by one, otherwise the behaviour should be the same.
"
curl -s -HX-API-Token:${TOKEN} -XPATCH -d'{"firmware":{"management":"device"}}' ${API_ENDPOINT}/collections/${COLLECTION} > /dev/null

step "Test with enabled firmware management (on device), current"
bin/fotaclient --horde-endpoint=${COAP_ENDPOINT} --scenario=noupdate --version=1.0.0
check_state "Current"

step "Set target to 2.0.0 on device"
description "
Set the target version to 2.0.0 by assigning the 2nd firmware image to the
device. The device's state should change from "Current" to "Pending" when the
new target version is set.
"
curl -s --fail -HX-API-Token:${TOKEN} -XPATCH -d"{\"firmware\":{\"targetFirmwareId\": \"${IMAGE2}\"}}"  ${API_ENDPOINT}/collections/${COLLECTION}/devices/${DEVICE} > /dev/null
check_state "Pending"

step "Test with enabled firmware management (on device), update"
bin/fotaclient --horde-endpoint=${COAP_ENDPOINT} --scenario=update --version=1.0.0
check_state "Completed"

step "Call back with 2.0.0 and ensure state is current"
bin/fotaclient --horde-endpoint=${COAP_ENDPOINT} --scenario=noupdate --version=2.0.0
check_state "Current"

step "Test with non-idle device"
description "
If the (LwM2M device) reports back a different state than "Idle" when queried by
the service it should go into an error state. The error state prevents further
updates of the firmware and must be reset before another attempt is made.
"
bin/fotaclient --horde-endpoint=${COAP_ENDPOINT} --scenario=nonidle --version=1.0.0
check_state "UpdateFailed"

step "Reset device state after non-idle state"
description "
Reset the device state by sending a delete request to the "fwerror" resource.
"
curl -s --fail -HX-API-Token:${TOKEN} -XDELETE ${API_ENDPOINT}/collections/${COLLECTION}/devices/${DEVICE}/fwerror

step "Test with missing resource on device"
bin/fotaclient --horde-endpoint=${COAP_ENDPOINT} --scenario=error --version=1.0.0
check_state "UpdateFailed"

step "Reset device state after resource error by setting target"
description "
The error state can also be reset by assigning a new target firmware to the
device. Ensure the error state is changed to "Pending" when the image is
assigned. The last time the device reported in it said 1.0.0 so we just set the
target to something different (2.0.0)
"
curl -s --fail -HX-API-Token:${TOKEN} -XPATCH -d"{\"firmware\": {\"targetFirmwareId\":\"${IMAGE2}\"}}" ${API_ENDPOINT}/collections/${COLLECTION}/devices/${DEVICE} > /dev/null
check_state "Pending"

step "Perform another update that fails"
description "
Start a new update but the next time the device reports back the version is
unchanged. This is typically if the download fails or the image is invalid. The
device should be flagged as "Reverted" the next time it calls back.
"
bin/fotaclient --horde-endpoint=${COAP_ENDPOINT} --scenario=update --version=0.9.0
check_state "Completed"

step "Call back with old version"
bin/fotaclient --horde-endpoint=${COAP_ENDPOINT} --scenario=noupdate --version=0.9.0
check_state "Reverted"

step "Reset device state"
curl -s --fail -HX-API-Token:${TOKEN} -XDELETE ${API_ENDPOINT}/collections/${COLLECTION}/devices/${DEVICE}/fwerror > /dev/null

step "Perform final update"
description "
Perform a final update that succeeds and reports back the correct version. The
device should get the final state Current when it has installed the update and
reported back a 2nd time.
"
bin/fotaclient --horde-endpoint=${COAP_ENDPOINT} --scenario=update --version=0.9.0
check_state "Completed"

step "Report up to date version to service"
description "
Report back the updated version to the service. The device state should be set
to Current when the update is complated.
"
bin/fotaclient --horde-endpoint=${COAP_ENDPOINT} --scenario=noupdate --version=2.0.0
check_state "Current"

section "Clean up"
step "Remove device"
curl -HX-API-Token:${TOKEN} -XDELETE ${API_ENDPOINT}/collections/${COLLECTION}/devices/${DEVICE} > /dev/null

step "Set firmware versions on collection"
description "
Since the firmware version is referenced on the collection we'll reset them to 0
to avoid issues with references.
"
curl -s --fail -HX-API-Token:${TOKEN} -XPATCH -d'{"firmware":{"targetFirmwareId":"0", "currentFirmwareId":"0"}}' ${API_ENDPOINT}/collections/${COLLECTION} > /dev/null

step "Remove firmware image 1"
curl -HX-API-Token:${TOKEN} -XDELETE ${API_ENDPOINT}/collections/${COLLECTION}/firmware/${IMAGE1} > /dev/null

step "Remove firmware image 2"
curl --fail  -HX-API-Token:${TOKEN} -XDELETE ${API_ENDPOINT}/collections/${COLLECTION}/firmware/${IMAGE2} > /dev/null

step "Remove collection"
curl --fail  -HX-API-Token:${TOKEN} -XDELETE ${API_ENDPOINT}/collections/${COLLECTION} > /dev/null

step "Remove IP allocation"
bin/ctrlh --endpoint=${MANAGEMENT} alloc rm --apnid=0 --nasid=0 --imsi=3

end_test
