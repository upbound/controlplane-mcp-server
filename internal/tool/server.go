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
	"k8s.io/client-go/kubernetes"

	"github.com/crossplane/crossplane-runtime/pkg/logging"

	"github.com/upbound/controlplane-mcp-server/internal/resource/pod"
)

// Server is a simple server for handling various tooling requests.
type Server struct {
	c   *kubernetes.Clientset
	log logging.Logger

	pod *pod.Pod
}

// Option modifies the underlying Server.
type Option func(*Server)

// WithLogging overrides the underlying Server.
func WithLogging(log logging.Logger) Option {
	return func(s *Server) {
		s.log = log
	}
}

// NewServer constructs a new Server.
func NewServer(c *kubernetes.Clientset, opts ...Option) *Server {
	s := &Server{
		c:   c,
		pod: pod.New(c),
		log: logging.NewNopLogger(),
	}

	for _, o := range opts {
		o(s)
	}

	return s
}
