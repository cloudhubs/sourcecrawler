package cfg

import (
	"sourcecrawler/app/db"
	"testing"
)

type labelTestCase struct {
	Name   string
	Root   db.Node
	Labels map[db.Node]db.ExecutionLabel
}

var cases = []func() labelTestCase{
	func() labelTestCase {
		endIf := &db.EndConditionalNode{}
		t := &db.FunctionNode{Child: endIf}
		f := &db.StatementNode{Child: endIf}
		endIf.SetParents(t)
		endIf.SetParents(f)
		root := &db.ConditionalNode{TrueChild: t, FalseChild: f}
		t.SetParents(root)
		f.SetParents(root)

		labels := make(map[db.Node]db.ExecutionLabel, 0)
		labels[root] = db.Must
		labels[t] = db.May
		labels[f] = db.May
		labels[endIf] = db.Must

		return labelTestCase{
			Name:   "if-else",
			Root:   root,
			Labels: labels,
		}
	},
	func() labelTestCase {
		endIf2 := &db.EndConditionalNode{}
		t2 := &db.StatementNode{Child: endIf2}
		f2 := &db.StatementNode{Child: endIf2}
		endIf2.SetParents(t2)
		endIf2.SetParents(f2)
		cond2 := &db.ConditionalNode{TrueChild: t2, FalseChild: f2}
		t2.SetParents(cond2)
		f2.SetParents(cond2)

		endIf1 := &db.EndConditionalNode{Child: cond2}
		t1 := &db.FunctionNode{}
		f1 := &db.FunctionNode{}
		endIf1.SetParents(t1)
		endIf1.SetParents(t1)
		cond1 := &db.ConditionalNode{TrueChild: t1, FalseChild: f1}
		t1.SetParents(cond1)
		f1.SetParents(cond1)

		root := &db.FunctionDeclNode{Child: cond1}
		labels := make(map[db.Node]db.ExecutionLabel, 0)
		labels[endIf2] = db.Must
		labels[t2] = db.May
		labels[f2] = db.May
		labels[cond2] = db.Must
		labels[endIf1] = db.Must
		labels[t1] = db.May
		labels[f1] = db.May
		labels[cond1] = db.Must
		labels[root] = db.Must

		return labelTestCase{
			Name:   "chained-if-else",
			Root:   root,
			Labels: labels,
		}
	},
}

func TestLabelNonCondNodes(t *testing.T) {
	tests := make([]labelTestCase, 0)
	for _, testCase := range cases {
		tests = append(tests, testCase())
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			LabelNonCondNodes(test.Root)
			traverse(test.Root, func(node db.Node) {
				if test.Labels[node] != node.GetLabel() {
					t.Errorf("%s had label %s, but %s was expected", node.GetProperties(), node.GetLabel(), test.Labels[node])
				}
			})
		})
	}
}
