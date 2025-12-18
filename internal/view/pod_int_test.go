// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of K9s

package view

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/derailed/k9s/internal"
	"github.com/derailed/k9s/internal/client"
	"github.com/derailed/k9s/internal/config"
	"github.com/derailed/k9s/internal/config/mock"
	"github.com/derailed/k9s/internal/dao"
	"github.com/derailed/k9s/internal/model"
	"github.com/derailed/k9s/internal/model1"
	"github.com/derailed/k9s/internal/ui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
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
	// Disable portForwardIndicator decorator to avoid factory nil error in tests
	// We only need to test selectRowByPath function, so decorator is not needed
	table.SetDecorateFn(nil)

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

	// Set a mock model so GetSelectedItem() works properly
	mockModel := &mockTableModelForTest{data: data}
	table.SetModel(mockModel)

	cdata := table.Update(data, false)
	table.UpdateUI(cdata, data)

	// Verify table is properly updated
	assert.Greater(t, table.GetRowCount(), 1, "Table should have rows")

	// Verify row IDs are properly set
	rowID, ok := table.GetRowID(1)
	require.True(t, ok, "Should be able to get row ID")
	assert.NotEmpty(t, rowID, "Row ID should not be empty")

	pod.selectRowByPath("default/pod2")

	selectedItem := table.GetSelectedItem()
	assert.Equal(t, "default/pod2", selectedItem, "pod2 should be selected")

	pod.selectRowByPath("default/nonexistent")

	selectedItem = table.GetSelectedItem()
	assert.Equal(t, "default/pod2", selectedItem)
}

// mockTableModelForTest is a minimal mock model for testing selectRowByPath
type mockTableModelForTest struct {
	data *model1.TableData
}

var _ ui.Tabular = (*mockTableModelForTest)(nil)

func (m *mockTableModelForTest) Empty() bool {
	return m.data == nil || m.data.RowCount() == 0
}

func (m *mockTableModelForTest) Peek() *model1.TableData {
	if m.data == nil {
		return model1.NewTableData(client.PodGVR)
	}
	return m.data
}

func (m *mockTableModelForTest) ClusterWide() bool       { return false }
func (m *mockTableModelForTest) GetNamespace() string    { return "default" }
func (m *mockTableModelForTest) SetNamespace(string)     {}
func (m *mockTableModelForTest) InNamespace(string) bool { return true }
func (m *mockTableModelForTest) Get(context.Context, string) (runtime.Object, error) {
	return nil, nil
}
func (m *mockTableModelForTest) SetInstance(string)                 {}
func (m *mockTableModelForTest) SetLabelSelector(labels.Selector)   {}
func (m *mockTableModelForTest) GetLabelSelector() labels.Selector  { return nil }
func (m *mockTableModelForTest) RowCount() int                      { return 1 }
func (m *mockTableModelForTest) Watch(context.Context) error        { return nil }
func (m *mockTableModelForTest) Refresh(context.Context) error      { return nil }
func (m *mockTableModelForTest) SetRefreshRate(time.Duration)       {}
func (m *mockTableModelForTest) AddListener(model.TableListener)    {}
func (m *mockTableModelForTest) RemoveListener(model.TableListener) {}
func (m *mockTableModelForTest) Delete(context.Context, string, *metav1.DeletionPropagation, dao.Grace) error {
	return nil
}
func (m *mockTableModelForTest) SetViewSetting(context.Context, *config.ViewSetting) {}

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
