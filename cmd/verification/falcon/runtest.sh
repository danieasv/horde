#!/bin/bash
TOKEN=$(curl -s -XPOST -d'{"resource":"/","write":true}' -HX-CONNECT-ID:1 http://localhost:8080/tokens | jq -r .token)
HOST=http://localhost:8080
RADIUS=127.0.0.1:1812
DATA=127.0.0.1:31415
./falcon --radius-ep=${RADIUS} -rest-api=${HOST} -data-ep=${DATA} -api-token=${TOKEN} --message-count 100 --message-interval 10ms --device-count 250 --remove-after-test
