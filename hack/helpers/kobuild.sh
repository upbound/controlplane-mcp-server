#!/usr/bin/env bash

set -e

_registry=$1
_ko_yaml_path=cluster/images/$2
_go_import_path=$3
_bare=$4

_version=${VERSION}
_git_build_tag=${GO_BUILD_TAG}
_push=true

unset GOOS
unset GOARCH

export KO_DOCKER_REPO=$_registry

if [[ -n "$_bare" ]]; then
    KO_DOCKER_REPO="$KO_DOCKER_REPO/$_bare"
fi
echo $KO_DOCKER_REPO

# Build the target OCI artifact for the given $_go_import_path.
# This script references the following attributes in order to accomplish this:
# * KO_CONFIG_PATH: This is pointed to the .ko.yaml for the respective artifact
#                   under cluster/images/___.
# *         --tags: This tells ko to tag the OCI image with the supplied tag
#                   instead of latest.
# *         --push: This tells ko to push the OCI images to the configured
#                   KO_DOCKER_REPO.
# *             -B: (short for --base-import-path). This tells ko to use the
#                   base path without MD5 hash after KO_DOCKER_REPO. This
#                   setting allows us to have more predictable names for the
#                   OCI images.
# Ref: https://ko.build/reference/ko_build/
IMAGE=$(KO_CONFIG_PATH=${_ko_yaml_path}/.ko.yaml $KO build $_go_import_path --tags=$_version --push=$_push -B)
echo $IMAGE
