#!/bin/sh
#
# Since there's quite a few command line parameters it's easier to maintain
# a shell script that is launched inside the Docker container than a
# CMD or EXEC block at the end of the Dockerfile
#
set -e
set -x
/horde-radius \
    --log-level debug \
    --log-type plain \
    --grpc-server-endpoint=${HORDE_RADIUS_ENDPOINT} \
    --radius-endpoint=${RADIUS_ENDPOINT} \
    --radius-shared-secret=${RADIUS_SHARED_SECRET}
