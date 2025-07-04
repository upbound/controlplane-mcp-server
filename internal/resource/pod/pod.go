/*
Package pod provides tool helpers for working with pods.
*/
package pod

import (
	"context"
	"encoding/json"
	"io"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
)

const (
	// defaultMaxEvents to return to the caller.
	defaultMaxEvents = 10
	// defaultMaxLogs to return to the caller.
	defaultMaxLogs = 10
)

// Pod provides methods for deriving details for pods in the configured
// controlplane.
type Pod struct {
	log logging.Logger
	cs  kubernetes.Interface

	// maximum number of events to return to the caller.
	maxEvents int
	// maximum number of log lines to return to the caller.
	maxLogLines int64
}

// Option modifies the underlying Pod.
type Option func(*Pod)

// WithLogger overrides the default loggger.
func WithLogger(log logging.Logger) Option {
	return func(p *Pod) {
		p.log = log
	}
}

// WithMaxEvents overrides the detault MaxEvents setting.
func WithMaxEvents(m int) Option {
	return func(p *Pod) {
		p.maxEvents = m
	}
}

// New constructs a new Pod.
func New(cs kubernetes.Interface, opts ...Option) *Pod {
	p := &Pod{
		cs:  cs,
		log: logging.NewNopLogger(),

		maxEvents:   defaultMaxEvents,
		maxLogLines: defaultMaxLogs,
	}

	for _, o := range opts {
		o(p)
	}

	return p
}

// GetLogs returns the logs from the supplied Pod up to the maximum number of
// log lines.
func (p *Pod) GetLogs(ctx context.Context, nn types.NamespacedName) ([]byte, error) {
	req := p.cs.CoreV1().Pods(nn.Namespace).GetLogs(nn.Name, &corev1.PodLogOptions{
		TailLines: ptr.To(p.maxLogLines),
	})
	logs, err := req.Stream(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read data from pod log stream")
	}
	defer func() {
		if err := logs.Close(); err != nil {
			p.log.Info("failed to close log stream", "error", err)
		}
	}()

	buf, err := io.ReadAll(logs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read pod log stream")
	}
	return buf, nil
}

// GetEvents returns the events for the correlated to the supplied pod up to
// the maximum number of events.
func (p *Pod) GetEvents(ctx context.Context, nn types.NamespacedName) ([]byte, error) {
	pod, err := p.cs.CoreV1().Pods(nn.Namespace).Get(ctx, nn.Name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to look up pod")
	}

	eventList, err := p.cs.CoreV1().Events(nn.Namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("involvedObject.name", pod.GetName()).String(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to look up events for pod")
	}

	seen := 0
	var sb strings.Builder

	for _, i := range eventList.Items {
		if seen >= p.maxEvents {
			break // we have enough
		}
		e := convert(i)
		s, err := e.string()
		if err != nil {
			p.log.Info("failed to marshal event", "error", err)
			// Skip this event in the event we have more useful data
			// to return.
			continue
		}

		sb.WriteString(s)
		seen++
	}

	// TODO(tnthornton) consider replacing this conversion.
	return []byte(sb.String()), nil
}

// event is a helper for logging corev1.Event details minus some attributes
// that we don't currently care about.
type event struct {
	Reason              string `json:"reason"`
	Message             string `json:"message"`
	EventTime           string `json:"eventTime"`
	Action              string `json:"action"`
	ReportingController string `json:"reportingController"`
	ReportingInstance   string `json:"reportingInstance"`
	Related             string `json:"related"`
	FirstTimestamp      string `json:"firstTimestamp"`
	LastTimestamp       string `json:"lastTimestamp"`
}

// convert the given corev1.Event into a event for cleaner log data.
func convert(e corev1.Event) *event {
	related := ""
	if e.Related != nil {
		related = e.Related.GroupVersionKind().String()
	}

	return &event{
		Reason:              e.Reason,
		Message:             e.Message,
		EventTime:           e.EventTime.String(),
		Action:              e.Action,
		ReportingController: e.ReportingController,
		ReportingInstance:   e.ReportingInstance,
		Related:             related,
		FirstTimestamp:      e.FirstTimestamp.String(),
		LastTimestamp:       e.LastTimestamp.String(),
	}
}

// stringify the event for sending back to the client.
func (e *event) string() (string, error) {
	b, err := json.Marshal(e)
	if err != nil {
		return "", errors.Wrap(err, "event content is broken")
	}
	return string(b), nil
}
