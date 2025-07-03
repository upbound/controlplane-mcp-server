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
