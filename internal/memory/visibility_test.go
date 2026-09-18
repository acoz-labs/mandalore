package memory

import (
	"fmt"
	"reflect"
	"testing"
)

func TestVisibilityExplicitConflictResolution(t *testing.T) {
	nodes := []visibilityNode{vc("c", nil, nil), vv("w", "withdraw", nil, []string{"c"}), vv("a", "restore", []string{"w"}, []string{"c"}), vv("b", "restore", []string{"w"}, []string{"c"})}
	nodes = append(nodes, vv("resolved", "restore", []string{"a", "b"}, []string{"c"}))
	got, err := evaluateVisibility(nodes)
	if err != nil || got.State != "visible" || !reflect.DeepEqual(got.VisibilityHeads, []string{"resolved"}) {
		t.Fatal(got, err)
	}
	nodes = append(nodes, vc("d", []string{"c"}, []string{"resolved"}), vc("e", []string{"c"}, []string{"resolved"}))
	got, err = evaluateVisibility(nodes)
	if err != nil || got.State != "content-conflict" {
		t.Fatal(got, err)
	}
}

func TestVisibilityDeepGraph(t *testing.T) {
	nodes := []visibilityNode{vc("c0", nil, nil)}
	for i := 1; i < 20000; i++ {
		nodes = append(nodes, vc(fmt.Sprint("c", i), []string{fmt.Sprint("c", i-1)}, nil))
	}
	nodes = append(nodes, vv("restore", "restore", nil, []string{"c19999"}))
	got, err := evaluateVisibility(nodes)
	if err != nil || got.State != "visible" {
		t.Fatal(got, err)
	}
}

func vc(id string, parents, observed []string) visibilityNode {
	return visibilityNode{ID: id, RecordID: "record-test", Kind: "content", Parents: parents, References: observed}
}
func vv(id, action string, parents, observed []string) visibilityNode {
	return visibilityNode{ID: id, RecordID: "record-test", Kind: action, Parents: parents, References: observed}
}

func TestVisibilityCausalStates(t *testing.T) {
	c := vc("c", nil, nil)
	w := vv("w", "withdraw", nil, []string{"c"})
	r := vv("r", "restore", []string{"w"}, []string{"c"})
	for _, tc := range []struct {
		name, state string
		nodes       []visibilityNode
	}{
		{"legacy", "visible", []visibilityNode{c}},
		{"withdrawal", "withdrawn", []visibilityNode{c, w}},
		{"correction cannot restore", "withdrawn", []visibilityNode{c, w, vc("d", []string{"c"}, []string{"w"})}},
		{"explicit restore", "visible", []visibilityNode{c, w, r}},
		{"aware correction", "visible", []visibilityNode{c, w, r, vc("d", []string{"c"}, []string{"r"})}},
		{"raced correction", "unreviewed-content", []visibilityNode{c, w, r, vc("d", []string{"c"}, []string{"w"})}},
		{"two restores", "visibility-conflict", []visibilityNode{c, w, r, vv("r2", "restore", []string{"w"}, []string{"c"})}},
		{"content conflict", "content-conflict", []visibilityNode{c, vc("a", []string{"c"}, nil), vc("b", []string{"c"}, nil)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := evaluateVisibility(tc.nodes)
			if err != nil || got.State != tc.state {
				t.Fatalf("got %+v, %v; want %s", got, err, tc.state)
			}
		})
	}
}

func TestVisibilityInvalidGraphs(t *testing.T) {
	c := vc("c", nil, nil)
	for name, nodes := range map[string][]visibilityNode{
		"empty":                nil,
		"duplicate":            {c, c},
		"missing":              {c, vv("w", "withdraw", nil, []string{"missing"})},
		"wrong kind":           {c, vv("w", "withdraw", []string{"c"}, []string{"c"})},
		"empty decision":       {c, vv("w", "withdraw", nil, nil)},
		"cross record":         {c, {ID: "w", RecordID: "record-other", Kind: "withdraw", References: []string{"c"}}},
		"combined cycle":       {vc("c", nil, []string{"r"}), vv("r", "restore", nil, []string{"c"})},
		"duplicate references": {c, vv("w", "withdraw", nil, []string{"c", "c"})},
		"unknown action":       {c, vv("w", "erase", nil, []string{"c"})},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := evaluateVisibility(nodes); err == nil {
				t.Fatal("accepted invalid graph")
			}
		})
	}
}

func TestVisibilityDeliveryPermutations(t *testing.T) {
	nodes := []visibilityNode{vc("c", nil, nil), vv("w", "withdraw", nil, []string{"c"}), vv("r", "restore", []string{"w"}, []string{"c"}), vc("d", []string{"c"}, []string{"w"}), vv("z", "restore", []string{"r"}, []string{"d"})}
	want, err := evaluateVisibility(nodes)
	if err != nil || want.State != "visible" {
		t.Fatal(want, err)
	}
	count := 0
	var permute func(int)
	permute = func(i int) {
		if i == len(nodes) {
			got, err := evaluateVisibility(nodes)
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("order-dependent: %+v %v", got, err)
			}
			count++
			return
		}
		for j := i; j < len(nodes); j++ {
			nodes[i], nodes[j] = nodes[j], nodes[i]
			permute(i + 1)
			nodes[i], nodes[j] = nodes[j], nodes[i]
		}
	}
	permute(0)
	if count != 120 {
		t.Fatal(count)
	}
}
