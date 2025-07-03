package pod

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
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
