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

func TestLabelNonCondNodes(t *testing.T) {
	cases := []func() labelTestCase{
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
		func() labelTestCase {
			labels := make(map[db.Node]db.ExecutionLabel, 0)

			leaf := &db.FunctionNode{}
			outerEndIf := &db.EndConditionalNode{Child: leaf}
			leaf.SetParents(outerEndIf)

			labels[leaf] = db.Must
			labels[outerEndIf] = db.Must

			// outer true branch
			trueEndIf := &db.EndConditionalNode{Child: outerEndIf}
			trueTrue := &db.FunctionNode{Child: trueEndIf}
			trueFalse := &db.StatementNode{Child: trueEndIf}
			trueEndIf.SetParents(trueTrue)
			trueEndIf.SetParents(trueFalse)
			trueCond := &db.ConditionalNode{TrueChild: trueTrue, FalseChild: trueFalse}
			trueTrue.SetParents(trueCond)
			trueFalse.SetParents(trueCond)
			trueNode2 := &db.StatementNode{Child: trueCond}
			trueNode1 := &db.FunctionNode{Child: trueNode2}
			trueNode2.SetParents(trueNode1)

			labels[trueEndIf] = db.May
			labels[trueTrue] = db.May
			labels[trueFalse] = db.May
			labels[trueCond] = db.May
			labels[trueNode2] = db.May
			labels[trueNode1] = db.May

			// outer false branch
			falseEndIf := &db.EndConditionalNode{Child: outerEndIf}
			falseTrue := &db.StatementNode{Child: falseEndIf}
			falseFalse := &db.StatementNode{Child: falseEndIf}
			falseEndIf.SetParents(falseTrue)
			falseEndIf.SetParents(falseFalse)
			falseNode1 := &db.ConditionalNode{TrueChild: falseTrue, FalseChild: falseFalse}
			falseTrue.SetParents(falseNode1)
			falseFalse.SetParents(falseNode1)

			outerEndIf.SetParents(trueEndIf)
			outerEndIf.SetParents(falseEndIf)

			labels[falseEndIf] = db.May
			labels[falseTrue] = db.May
			labels[falseFalse] = db.May
			labels[falseNode1] = db.May

			root := &db.ConditionalNode{TrueChild: trueNode1, FalseChild: falseNode1}
			labels[root] = db.Must

			return labelTestCase{
				Name:   "nested-if-else-extra",
				Root:   root,
				Labels: labels,
			}
		},
	}

	for _, testCase := range cases {
		test := testCase()
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
