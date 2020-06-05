#!/bin/sh
#
# Since there's quite a few command line parameters it's easier to maintain
# a shell script that is launched inside the Docker container than a
# CMD or EXEC block at the end of the Dockerfile
#
set -e
set -x
/horde-udp \
    --log-level debug \
    --log-type plain \
    --grpc-server-endpoint=${RXTX_GRPC_ENDPOINT} \
    --udp-ports=${LISTEN_PORTS} \
    --udp-apnid=0 \
    --udp-nasid=0 \
    --udp-listen-address=0.0.0.0
