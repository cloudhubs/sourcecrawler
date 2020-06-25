package cfg

import (
	"sourcecrawler/app/db"
	"testing"
)

type labelTestCase struct {
	Name   string
	Root   db.Node
	Labels map[db.Node]db.ExecutionLabel
	Regexs map[int]string
}

func TestLabelNonCondNodes(t *testing.T) {
	cases := []func() labelTestCase{
		func() labelTestCase {
			endIf := &db.EndConditionalNode{}
			t := &db.FunctionNode{Child: endIf}
			f := &db.FunctionNode{Child: endIf}
			endIf.SetParents(t)
			endIf.SetParents(f)
			root := &db.ConditionalNode{TrueChild: t, FalseChild: f}
			t.SetParents(root)
			f.SetParents(root)

			labels := make(map[db.Node]db.ExecutionLabel)
			labels[root] = db.Must
			labels[t] = db.May
			labels[f] = db.May
			labels[endIf] = db.Must

			return labelTestCase{
				Name:   "if-else",
				Root:   root,
				Labels: labels,
				Regexs: make(map[int]string),
			}
		},
		func() labelTestCase {
			endIf2 := &db.EndConditionalNode{}
			t2 := &db.FunctionNode{Child: endIf2}
			f2 := &db.FunctionNode{Child: endIf2}
			endIf2.SetParents(t2)
			endIf2.SetParents(f2)
			cond2 := &db.ConditionalNode{TrueChild: t2, FalseChild: f2}
			t2.SetParents(cond2)
			f2.SetParents(cond2)

			endIf1 := &db.EndConditionalNode{Child: cond2}
			t1 := &db.FunctionNode{}
			f1 := &db.FunctionNode{}
			endIf1.SetParents(t1)
			endIf1.SetParents(f1)
			cond1 := &db.ConditionalNode{TrueChild: t1, FalseChild: f1}
			t1.SetParents(cond1)
			f1.SetParents(cond1)

			root := &db.FunctionDeclNode{Child: cond1}
			labels := make(map[db.Node]db.ExecutionLabel)
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
				Regexs: make(map[int]string),
			}
		},
		func() labelTestCase {
			labels := make(map[db.Node]db.ExecutionLabel)

			leaf := &db.FunctionNode{}
			outerEndIf := &db.EndConditionalNode{Child: leaf}
			leaf.SetParents(outerEndIf)

			labels[leaf] = db.Must
			labels[outerEndIf] = db.Must

			// outer true branch
			trueEndIf := &db.EndConditionalNode{Child: outerEndIf}
			trueTrue := &db.FunctionNode{Child: trueEndIf}
			trueFalse := &db.FunctionNode{Child: trueEndIf}
			trueEndIf.SetParents(trueTrue)
			trueEndIf.SetParents(trueFalse)
			trueCond := &db.ConditionalNode{TrueChild: trueTrue, FalseChild: trueFalse}
			trueTrue.SetParents(trueCond)
			trueFalse.SetParents(trueCond)
			trueNode2 := &db.FunctionNode{Child: trueCond}
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
			falseTrue := &db.FunctionNode{Child: falseEndIf}
			falseFalse := &db.FunctionNode{Child: falseEndIf}
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
				Regexs: make(map[int]string),
			}
		},
		func() labelTestCase {
			endIf := &db.EndConditionalNode{}
			root := &db.ConditionalNode{TrueChild: endIf, FalseChild: endIf}
			endIf.SetParents(endIf)
			endIf.SetParents(endIf)

			labels := make(map[db.Node]db.ExecutionLabel)
			labels[root] = db.Must
			labels[endIf] = db.Must

			return labelTestCase{
				Name:   "dead-if-else",
				Root:   root,
				Labels: labels,
				Regexs: make(map[int]string),
			}
		},
		func() labelTestCase {
			end := &db.EndConditionalNode{}
			t1 := &db.FunctionNode{Child: end}
			f1 := &db.StatementNode{
				LogRegex:   "this is a log message: .*",
				LineNumber: 67,
				Child:      end,
			}
			end.SetParents(t1)
			end.SetParents(f1)

			root := &db.ConditionalNode{TrueChild: t1, FalseChild: f1}
			labels := make(map[db.Node]db.ExecutionLabel)
			labels[end] = db.Must
			labels[t1] = db.MustNot
			labels[f1] = db.Must
			labels[root] = db.Must

			regexs := make(map[int]string)
			regexs[67] = "this is a log message: .*"

			return labelTestCase{
				Name:   "log-if-else",
				Root:   root,
				Labels: labels,
				Regexs: regexs,
			}
		},
	}

	for _, testCase := range cases {
		test := testCase()
		t.Run(test.Name, func(t *testing.T) {
			LabelParentNodes(test.Root, test.Regexs)
			traverse(test.Root, func(node db.Node) {
				if test.Labels[node] != node.GetLabel() {
					t.Errorf(
						"%s had label %s, but %s was expected",
						node.GetProperties(),
						node.GetLabel().String(),
						test.Labels[node].String(),
					)
				}
			})
		})
	}
}
