package pod

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetLogs(t *testing.T) {
	type args struct {
		nn types.NamespacedName
		cs kubernetes.Interface
	}
	type want struct {
		res []byte
		err error
	}

	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"Success": {
			reason: "If the pod is available, we shouldn't fail to get the logs",
			args: args{
				cs: fake.NewClientset(),
				nn: types.NamespacedName{
					Namespace: "default",
					Name:      "pod-1",
				},
			},
			want: want{
				res: []byte("fake logs"),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			p := New(tc.args.cs)
			got, err := p.GetLogs(context.Background(), tc.args.nn)

			if diff := cmp.Diff(tc.want.err, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("\n%s\nGetLogs(...): -want err, +got err:\n%s", tc.reason, diff)
			}

			if diff := cmp.Diff(tc.want.res, got); diff != "" {
				t.Errorf("\n%s\nGetLogs(...): -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}

func TestGetEvents(t *testing.T) {
	type args struct {
		nn types.NamespacedName
		cs kubernetes.Interface
	}
	type want struct {
		numEvents int
		err       error
	}

	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"NoEvents": {
			reason: "If the pod is available but there are no events, and empty string should be returned.",
			args: args{
				cs: fake.NewClientset(&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-1",
						Namespace: "default",
					},
				}),
				nn: types.NamespacedName{
					Namespace: "default",
					Name:      "pod-1",
				},
			},
			want: want{
				numEvents: 0,
			},
		},
		"SomeEvents": {
			reason: "If the pod is available and there are a couple of events, those events are returned.",
			args: args{
				cs: fake.NewClientset(&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-1",
						Namespace: "default",
					},
				}, &corev1.Event{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "event-1",
					},
					InvolvedObject: corev1.ObjectReference{
						Name: "pod-1",
					},
					Reason:  "some reason",
					Message: "some message",
				},
					&corev1.Event{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "default",
							Name:      "event-2",
						},
						InvolvedObject: corev1.ObjectReference{
							Name: "pod-1",
						},
						Reason:  "some other reason",
						Message: "some other message",
					}),
				nn: types.NamespacedName{
					Namespace: "default",
					Name:      "pod-1",
				},
			},
			want: want{
				numEvents: 2,
			},
		},
		"MoreEventThanMax": {
			reason: "If the pod is available and there are more than the maximum number of events, only the max number is returned.",
			args: args{
				cs: func() kubernetes.Interface {
					objs := make([]runtime.Object, 0)
					for _, e := range getEvents(10) {
						objs = append(objs, e)
					}

					objs = append(objs, &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "pod-1",
							Namespace: "default",
						},
					})

					return fake.NewClientset(objs...)
				}(),
				nn: types.NamespacedName{
					Namespace: "default",
					Name:      "pod-1",
				},
			},
			want: want{
				numEvents: 10,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			p := New(tc.args.cs)
			got, err := p.GetEvents(context.Background(), tc.args.nn)

			if diff := cmp.Diff(tc.want.err, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("\n%s\nGetLogs(...): -want err, +got err:\n%s", tc.reason, diff)
			}

			if diff := cmp.Diff(tc.want.numEvents, count(t, got)); diff != "" {
				t.Errorf("\n%s\nGetLogs(...): -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}

func getEvents(n int) []*corev1.Event {
	list := make([]*corev1.Event, 0)

	for i := range n {
		list = append(list, &corev1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      fmt.Sprintf("event-%d", i),
			},
			InvolvedObject: corev1.ObjectReference{
				Name: "pod-1",
			},
			Reason:  "some other reason",
			Message: "some other message",
		},
		)
	}

	return list
}

// count the number of individual events in the JSON stream.
func count(t *testing.T, el []byte) int {
	t.Helper()

	d := json.NewDecoder(bytes.NewReader(el))

	events := make([]event, 0)

	for d.More() {
		var e event
		if err := d.Decode(&e); err != nil {
			t.Fatalf("failed to decode event in stream: %v", err)
		}
		events = append(events, e)
	}

	return len(events)
}
