#!/bin/sh
#
# Since there's quite a few command line parameters it's easier to maintain
# a shell script that is launched inside the Docker container than a
# CMD or EXEC block at the end of the Dockerfile
#
set -e
set -x
/messagereceiver \
    --coap=0.0.0.0:4712 \
    --http=0.0.0.0:8282 \
    --udp=0.0.0.0:4711 \
    --upstream-coap=${UPSTREAM_COAP} \
    --upstream-udp=${UPSTREAM_UDP}
