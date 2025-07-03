package tool

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"k8s.io/apimachinery/pkg/types"
)

const getPodLogs = "get_pod_logs"

// GetPodLogs creates a new mcp.Tool for retrieving pods logs from the matching
// pod details provided as parameters.
func GetPodLogs() mcp.Tool {
	return mcp.NewTool(getPodLogs,
		mcp.WithDescription(`
Read the logs of the given container of the given Kubernetes pod in the given namespace.
`),
		mcp.WithString("namespace",
			mcp.Required(),
			mcp.Description("The Kubernetes namespace of the pod"),
		),
		mcp.WithString("pod",
			mcp.Required(),
			mcp.Description("The name of the Kubernetes pod"),
		),
		mcp.WithString("container",
			mcp.Description("The name of the container of the pod whose logs are being read"),
		),
	)
}

// GetPodLogsHander handles tool requests to retrieve pods logs.
func (s *Server) GetPodLogsHander(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log := s.log.WithValues("handler", getPodLogs)
	log.Debug("received request")

	// helper functions for type-safe argument access
	ns, err := req.RequireString("namespace")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	name, err := req.RequireString("pod")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// container, err := request.RequireString("container")
	// if err != nil {
	// 	return mcp.NewToolResultError(err.Error()), nil
	// }

	logs, err := s.pod.GetLogs(ctx, types.NamespacedName{Namespace: ns, Name: name})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(string(logs)), nil
}
