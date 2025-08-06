# controlplane-mcp-server
[![CI](https://github.com/upbound/controlplane-mcp-server/actions/workflows/ci.yaml/badge.svg)](https://github.com/upbound/controlplane-mcp-server/actions/workflows/ci.yaml)
[![Slack](https://img.shields.io/badge/slack-upbound_crossplane-purple?logo=slack)](https://crossplane.slack.com/archives/C01TRKD4623)
[![GitHub release](https://img.shields.io/github/release/upbound/controlplane-mcp-server/all.svg)](https://github.com/upbound/controlplane-mcp-server/releases)

## Features

* Read Events: Look up events corresponding to the supplied pod.
* Read Pod Logs: Look up logs corresponding to the supplied pod.

## Example Usage with Intelligent Function
```yaml
apiVersion: pkg.crossplane.io/v1beta1
kind: DeploymentRuntimeConfig
metadata:
  name: ctp-mcp
spec:
  serviceAccountTemplate:
    metadata:
      # We need to provide additional permissions to the function. In order to
      # do that we create a deterministic ServiceAccount name.
      name: function-pod-analyzer
  deploymentTemplate:
    spec:
      selector: {}
      template:
        spec:
          containers:
          - name: package-runtime
            args:
            - --debug
            # Fine for local development (using crossplane render). Not fine
            # when integrated with Crossplane.
            # - --insecure
            env:
            # Inform the function of the CTP1 MCP Server.
            # transport: http-stream indicates that we'll communicate with the
            # MCP server over StreamableHTTP.
            - name: MCP_SERVER_TOOL_CTP1_TRANSPORT
              value: http-stream
            # baseURL indicates which address and endpoint to reach out to for
            # tooling.
            - name: MCP_SERVER_TOOL_CTP1_BASEURL
              value: http://localhost:8080/mcp
          - name: controlplane-mcp-server
            image: xpkg.upbound.io/upbound/controlplane-mcp-server:v0.1.0
            args:
            - --debug
```

Required Permissions for the function:
```yaml
---
# log-and-event-reader provides sufficient yet narrow scoped permissions for
# reading pod logs and events related to the pod.
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: log-and-event-reader
rules:
# controlplane-mcp-server needs get/list on pods, pods/log, and events
# in order to retrieve information for analysis.
- apiGroups:
  - ""
  resources:
  - events
  - pods
  - pods/log
  verbs:
  - get
  - list
---
# Bind the above ClusterRole to the function's service account.
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: log-and-event-reader
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: log-and-event-reader
subjects:
- kind: ServiceAccount
  name: function-pod-analyzer
  namespace: crossplane-system
```

Function Spec:
```yaml
---
apiVersion: pkg.crossplane.io/v1beta1
kind: Function
metadata:
  name: function-claude
spec:
  package: xpkg.upbound.io/upbound/function-claude:v0.1.0
  runtimeConfigRef:
    name: ctp-mcp
```

## Available Tools

1. get_pod_logs

Read the logs of the given container of the given Kubernetes pod in the given namespace.

Parameters:

* namespace (string, required): The Kubernetes namespace of the pod
* pod (string, required): The name of the Kubernetes pod
* container (string): The name of the container of the pod whose logs are being
read

2. get_pod_events

Read the events of the given Kubernetes pod in the given namespace.

Parameters:
* namespace (string, required): The Kubernetes namespace of the pod
* pod (string, required): The name of the Kubernetes pod
* container (string): The name of the container of the pod whose logs are being
read
