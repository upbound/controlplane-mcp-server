#!/usr/bin/env bash
set -o errexit

_net_dir="./cluster/local/config/networking"

# create registry container unless it already exists
reg_name='ctlptl-registry'
reg_port='5001'

# create a cluster with the local registry enabled in containerd
# TODO(tnthornton) break this into multiple deploys depending on
# what we want to validate. e.g. 1 node vs 3 node.
cat <<EOF | ${KIND} create cluster --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: ${KIND_CLUSTER_NAME:-local-dev}
networking:
  apiServerAddress: '127.0.0.1'
  apiServerPort: 6443
  disableDefaultCNI: false
  podSubnet: "192.168.0.0/16"
nodes:
  - role: control-plane
    extraPortMappings:
      - containerPort: 30000
        hostPort: 5432
    extraMounts:
      - containerPath: /var/lib/kubelet/config.json
        hostPath: /etc/docker/config.json
    kubeadmConfigPatches:
      - |
        kind: InitConfiguration
        nodeRegistration:
          kubeletExtraArgs:
            authorization-mode: "AlwaysAllow"
containerdConfigPatches:
- |-
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."localhost:${reg_port}"]
    endpoint = ["https://${reg_name}:5000"]
  [plugins."io.containerd.grpc.v1.cri".registry.configs."${reg_name}:5000".tls]
      insecure_skip_verify = true
EOF

# connect the registry to the cluster network if not already connected
if [ "$(docker inspect -f='{{json .NetworkSettings.Networks.kind}}' "${reg_name}")" = 'null' ]; then
  docker network connect "kind" "${reg_name}"
fi

# Document the local registry
# https://github.com/kubernetes/enhancements/tree/master/keps/sig-cluster-lifecycle/generic/1755-communicating-a-local-registry
cat <<EOF | ${KUBECTL} apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: local-registry-hosting
  namespace: kube-public
data:
  localRegistryHosting.v1: |
    host: "localhost:${reg_port}"
    hostFromContainerRuntime: "ctlptl-registry:5000"
    hostFromClusterNetwork: "ctlptl-registry:5000"
    help: "https://kind.sigs.k8s.io/docs/user/local-registry/"
EOF
