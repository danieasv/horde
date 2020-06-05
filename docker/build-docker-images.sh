#!/bin/bash
set -e

if [ "${VERSION}" == "" ]; then
    echo VERSION environment variable must be set
    exit 1
fi

if [ "${VERSION}" == "develop" ]; then
    cp ../bin/horde.linux core
    cp ../bin/magpie.linux datastore
    cp ../bin/messagereceiver.linux messagereceiver
    cp ../bin/horde-radius.linux horde-radius
    cp ../bin/horde-udp.linux horde-udp
    cp ../bin/horde-coap.linux horde-coap
else
    # Build docker images and tag with version
    mkdir tmpbin
    cp ../release/archives/${VERSION}/${VERSION}-horde_amd64-linux.zip tmpbin/linux.zip
    cd tmpbin
    unzip linux.zip
    rm linux.zip
    cd ..
    cp tmpbin/horde.linux core
    cp tmpbin/magpie.linux datastore
    cp tmpbin/messagereceiver.linux datastore
    cp tmpbin/horde-radius.linux horde-radius
    cp tmpbin/horde-udp.linux horde-udp
    cp tmpbin/horde-coap.linux horde-coap
    rm -fR tmpbin
fi

docker build core --tag horde-core:${VERSION}
docker build datastore --tag horde-datastore:${VERSION}
docker build messagereceiver --tag horde-messagereceiver:${VERSION}
docker build horde-radius --tag horde-radius:${VERSION}
docker build horde-udp --tag horde-udp:${VERSION}
docker build horde-coap --tag horde-coap:${VERSION}
