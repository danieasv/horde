#!/bin/sh
#
# Since there's quite a few command line parameters it's easier to maintain
# a shell script that is launched inside the Docker container than a
# CMD or EXEC block at the end of the Dockerfile
#
set -e
set -x
/horde-coap \
    --log-level debug \
    --log-type plain \
    --grpc-server-endpoint=${RXTX_GRPC_ENDPOINT} \
    --coap-apnid=0 \
    --coap-nasid=0 \
    --coap-endpoint=${COAP_ENDPOINT}
