#!/usr/bin/env bash

set -e

docker ps -f name=ctlptl-registry --format '{{.Names}}' | grep ctlptl-registry
exit $?