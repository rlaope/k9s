// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of K9s

package view

import (
	"context"
	"strings"
	"testing"

	"github.com/derailed/k9s/internal"
	"github.com/derailed/k9s/internal/client"
	"github.com/derailed/k9s/internal/config/mock"
	"github.com/derailed/k9s/internal/model1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func newStr(s string) *string {
	return &s
}

func TestComputeShellArgs(t *testing.T) {
	uu := map[string]struct {
		fqn, co, os string
		cfg         *genericclioptions.ConfigFlags
		e           string
	}{
		"config": {
			fqn: "fred/blee",
			co:  "c1",
			os:  "darwin",
			cfg: &genericclioptions.ConfigFlags{
				KubeConfig: newStr("coolConfig"),
			},
			e: "exec -it -n fred blee --kubeconfig coolConfig -c c1 -- sh -c " + shellCheck,
		},

		"no-config": {
			fqn: "fred/blee",
			co:  "c1",
			os:  "linux",
			e:   "exec -it -n fred blee -c c1 -- sh -c " + shellCheck,
		},

		"empty-config": {
			fqn: "fred/blee",
			cfg: new(genericclioptions.ConfigFlags),
			e:   "exec -it -n fred blee -- sh -c " + shellCheck,
		},

		"single-container": {
			fqn: "fred/blee",
			os:  "linux",
			cfg: new(genericclioptions.ConfigFlags),
			e:   "exec -it -n fred blee -- sh -c " + shellCheck,
		},

		"windows": {
			fqn: "fred/blee",
			co:  "c1",
			os:  windowsOS,
			cfg: new(genericclioptions.ConfigFlags),
			e:   "exec -it -n fred blee -c c1 -- powershell",
		},

		"full": {
			fqn: "fred/blee",
			co:  "c1",
			os:  windowsOS,
			cfg: &genericclioptions.ConfigFlags{
				KubeConfig:  newStr("coolConfig"),
				Context:     newStr("coolContext"),
				BearerToken: newStr("coolToken"),
			},
			e: "exec -it -n fred blee --kubeconfig coolConfig --context coolContext --token coolToken -c c1 -- powershell",
		},
	}

	for k := range uu {
		u := uu[k]
		t.Run(k, func(t *testing.T) {
			args := computeShellArgs(u.fqn, u.co, u.cfg, u.os)
			assert.Equal(t, u.e, strings.Join(args, " "))
		})
	}
}

func TestPodSelectRowByPath(t *testing.T) {
	po := NewPod(client.PodGVR)
	ctx := context.WithValue(context.Background(), internal.KeyApp, NewApp(mock.NewMockConfig(t)))
	require.NoError(t, po.Init(ctx))

	pod, ok := po.(*Pod)
	require.True(t, ok, "NewPod should return *Pod")

	table := pod.GetTable()

	data := model1.NewTableDataWithRows(
		client.PodGVR,
		model1.Header{
			model1.HeaderColumn{Name: "NAMESPACE"},
			model1.HeaderColumn{Name: "NAME"},
		},
		model1.NewRowEventsWithEvts(
			model1.RowEvent{
				Row: model1.Row{
					ID:     "default/pod1",
					Fields: model1.Fields{"default", "pod1"},
				},
			},
			model1.RowEvent{
				Row: model1.Row{
					ID:     "default/pod2",
					Fields: model1.Fields{"default", "pod2"},
				},
			},
			model1.RowEvent{
				Row: model1.Row{
					ID:     "default/pod3",
					Fields: model1.Fields{"default", "pod3"},
				},
			},
		),
	)

	cdata := table.Update(data, false)
	table.UpdateUI(cdata, data)

	pod.selectRowByPath("default/pod2")

	selectedItem := table.GetSelectedItem()
	assert.Equal(t, "default/pod2", selectedItem)

	pod.selectRowByPath("default/nonexistent")

	selectedItem = table.GetSelectedItem()
	assert.Equal(t, "default/pod2", selectedItem)
}

func TestPodSelectRowByPathWithEmptyTable(t *testing.T) {
	po := NewPod(client.PodGVR)
	ctx := context.WithValue(context.Background(), internal.KeyApp, NewApp(mock.NewMockConfig(t)))
	require.NoError(t, po.Init(ctx))

	pod, ok := po.(*Pod)
	require.True(t, ok, "NewPod should return *Pod")

	pod.selectRowByPath("default/pod1")

	table := pod.GetTable()
	selectedItem := table.GetSelectedItem()
	assert.Empty(t, selectedItem)
}
