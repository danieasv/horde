#!/bin/sh
#
# Since there's quite a few command line parameters it's easier to maintain
# a shell script that is launched inside the Docker container than a
# CMD or EXEC block at the end of the Dockerfile
#
set -e
set -x
/horde \
    --log-level=debug \
    --log-type=plain \
    --connect-enabled=true \
    --connect-client-id=${CONNECT_CLIENT_ID} \
    --connect-password=${CONNECT_CLIENT_SECRET} \
    --connect-host=${CONNECT_HOST} \
    --db-create-schema=true \
    --db-type=${DB_TYPE} \
    --db-connection-string=${DB_CONNECTION_STRING} \
    --github-client-id=${GITHUB_CLIENT_ID}   \
    --github-client-secret=${GITHUB_CLIENT_SECRET} \
    --github-db-connection-string=${DB_CONNECTION_STRING} \
    --github-db-driver=${DB_TYPE} \
    --launch-data-storage=false \
    --http-endpoint=${HTTP_ENDPOINT} \
    --http-inline-requestlog \
    --monitoring-endpoint=${MONITORING_ENDPOINT} \
    --grpc-data-store-server-endpoint=${DATASTORE_ENDPOINT} \
    --connect-login-target=http://localhost:8181/ \
    --connect-logout-target=http://localhost:8181/ \
    --enable-local-outputs=true \
    --worker-id=${WORKER_ID} \
    --data-center-id=${DATACENTER_ID} \
    --embedded-radius=false \
    --radius-grpc-endpoint=${RADIUS_GRPC_ENDPOINT} \
    --embedded-listener=false \
    --rx-tx-grpc-endpoint=${RXTX_GRPC_ENDPOINT} \
    --fota-firmware-endpoint=coap://127.0.0.1:5683/fw \
    --fota-lwm2m-timeout=5s \
    --fota-download-timeout=30s \
    --fota-lwm2m-poll-interval=1s \
    --management-endpoint=${MANAGEMENT_ENDPOINT}

    ## Local outputs are enabled for testing. The APN runs on an internal network.
