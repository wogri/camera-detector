#!/bin/bash -e -x

# hass.sh - A script to manage Home Assistant service
# requests a single parameter: on or off.

source ./env
URL=$HASS/api/services/input_boolean/turn_$1

curl -X POST \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"entity_id\": \"$ENTITY_ID\"}" \
    $URL
