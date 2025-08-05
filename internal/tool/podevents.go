// /*
// Copyright 2025 The Upbound Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

package tool

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"k8s.io/apimachinery/pkg/types"
)

const getPodEvents = "get_pod_events"

// GetPodEvents creates a new mcp.Tool for retrieving pods events from the matching
// pod details provided as parameters.
func GetPodEvents() mcp.Tool {
	return mcp.NewTool(getPodEvents,
		mcp.WithDescription(`
Read the events of the given Kubernetes pod in the given namespace.
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

// GetPodEventsHander handles tool requests to retrieve pod events.
func (s *Server) GetPodEventsHander(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log := s.log.WithValues("handler", getPodEvents)
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

	events, err := s.pod.GetEvents(ctx, types.NamespacedName{Namespace: ns, Name: name})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(string(events)), nil
}
