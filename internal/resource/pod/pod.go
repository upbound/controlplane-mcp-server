package pod

import (
	"context"
	"io"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
)

type Pod struct {
	log logging.Logger
	c   *kubernetes.Clientset
}

type Option func(*Pod)

func WithLogger(log logging.Logger) Option {
	return func(p *Pod) {
		p.log = log
	}
}

func New(c *kubernetes.Clientset, opts ...Option) *Pod {
	p := &Pod{
		c:   c,
		log: logging.NewNopLogger(),
	}

	for _, o := range opts {
		o(p)
	}

	return p
}

func (p *Pod) GetLogs(ctx context.Context, nn types.NamespacedName) ([]byte, error) {
	req := p.c.CoreV1().Pods(nn.Namespace).GetLogs(nn.Name, &v1.PodLogOptions{})
	logs, err := req.Stream(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read data from pod log stream")
	}

	defer logs.Close()

	buf, err := io.ReadAll(logs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read pod log stream")
	}
	return buf, nil
}
