package chartresolver

import (
	"encoding/json"
	"time"

	chart "helm.sh/helm/v4/pkg/chart/v2"
	"helm.sh/helm/v4/pkg/chart/common"
)

func MarshalChart(chart *chart.Chart) ([]byte, error) {
	tree := treeFromChart(chart)
	return json.Marshal(tree)
}

func UnmarshalChart(bytes []byte) (*chart.Chart, error) {
	tree := &ChartTree{}
	if err := json.Unmarshal(bytes, tree); err != nil {
		return nil, err
	}
	return chartFromTree(tree), nil
}

// ChartTree is a tree containing a chart and its subcharts (recursively),
// both in public fields so that they are respected during marshaling.
type ChartTree struct {
	Chart    *chart.Chart `json:"chart,omitempty"`
	SubTrees []*ChartTree `json:"subTrees,omitempty"`
}

func treeFromChart(chart *chart.Chart) *ChartTree {
	tree := &ChartTree{
		Chart: chart,
	}

	subCharts := chart.Dependencies()
	if len(subCharts) > 0 {
		tree.SubTrees = make([]*ChartTree, len(subCharts))
		for i := range subCharts {
			tree.SubTrees[i] = treeFromChart(subCharts[i])
		}
	}

	return tree
}

func chartFromTree(tree *ChartTree) *chart.Chart {
	ch := tree.Chart

	if len(tree.SubTrees) > 0 {
		subCharts := make([]*chart.Chart, len(tree.SubTrees))
		for i := range tree.SubTrees {
			subCharts[i] = chartFromTree(tree.SubTrees[i])
		}
		ch.AddDependency(subCharts...)
	}

	return ch
}

// stripModTimes zeroes out all ModTime fields in the chart recursively.
// The omitzero json tag on these fields means non-zero times are serialised
// but zero times are omitted, so after a JSON round-trip non-zero ModTimes
// become zero. Zeroing them upfront makes fresh and cached charts consistent.
func stripModTimes(ch *chart.Chart) {
	ch.ModTime = time.Time{}
	ch.SchemaModTime = time.Time{}
	for _, f := range ch.Templates {
		stripFileModTime(f)
	}
	for _, f := range ch.Files {
		stripFileModTime(f)
	}
	for _, dep := range ch.Dependencies() {
		stripModTimes(dep)
	}
}

func stripFileModTime(f *common.File) {
	if f != nil {
		f.ModTime = time.Time{}
	}
}
