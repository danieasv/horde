#!/bin/sh
#
# Since there's quite a few command line parameters it's easier to maintain
# a shell script that is launched inside the Docker container than a
# CMD or EXEC block at the end of the Dockerfile
#
set -e
set -x
/magpie \
    --log-level debug \
    --log-type plain \
    --monitoring-endpoint=${MONITORING_ENDPOINT} \
    --grpc-endpoint=${GRPC_ENDPOINT} \
    --sql-connection-string=${DB_CONNECTION_STRING} \
    --sql-create-schema=true \
    --sql-type=${DB_TYPE}
