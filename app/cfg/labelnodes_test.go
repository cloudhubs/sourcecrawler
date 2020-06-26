package cfg

import (
	"sourcecrawler/app/db"
	"sourcecrawler/app/model"
	"testing"
)

type labelTestCase struct {
	Name   string
	Root   db.Node
	Leaf   db.Node
	Labels map[db.Node]db.ExecutionLabel
	Logs   []model.LogType
}

func TestLabelNonCondNodes(t *testing.T) {
	cases := []func() labelTestCase{
		func() labelTestCase {
			leaf := &db.StatementNode{}
			endIf := &db.EndConditionalNode{Child: leaf}
			leaf.SetParents(endIf)
			t := &db.FunctionNode{Child: endIf, LineNumber: 2}
			f := &db.StatementNode{Child: endIf, LineNumber: 3}
			endIf.SetParents(t)
			endIf.SetParents(f)
			root := &db.ConditionalNode{TrueChild: t, FalseChild: f, LineNumber: 1}
			t.SetParents(root)
			f.SetParents(root)

			labels := make(map[db.Node]db.ExecutionLabel)
			labels[root] = db.Must
			labels[t] = db.May
			labels[f] = db.May
			labels[endIf] = db.Must
			labels[leaf] = db.Must

			return labelTestCase{
				Name:   "if-else",
				Root:   root,
				Leaf:   leaf,
				Labels: labels,
				Logs:   make([]model.LogType, 0),
			}
		},
		func() labelTestCase {
			leaf := &db.StatementNode{}
			endIf2 := &db.EndConditionalNode{Child: leaf}
			leaf.SetParents(endIf2)
			t2 := &db.StatementNode{Child: endIf2}
			f2 := &db.StatementNode{Child: endIf2}
			endIf2.SetParents(t2)
			endIf2.SetParents(f2)
			cond2 := &db.ConditionalNode{TrueChild: t2, FalseChild: f2}
			t2.SetParents(cond2)
			f2.SetParents(cond2)

			endIf1 := &db.EndConditionalNode{Child: cond2}
			cond2.SetParents(endIf1)
			t1 := &db.FunctionNode{}
			f1 := &db.FunctionNode{}
			endIf1.SetParents(t1)
			endIf1.SetParents(f1)
			cond1 := &db.ConditionalNode{TrueChild: t1, FalseChild: f1}
			t1.SetParents(cond1)
			f1.SetParents(cond1)

			root := &db.FunctionDeclNode{Child: cond1}
			cond1.SetParents(root)
			labels := make(map[db.Node]db.ExecutionLabel, 0)
			labels[leaf] = db.Must
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
				Leaf:   leaf,
				Labels: labels,
				Logs:   make([]model.LogType, 0),
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
			trueNode2 := &db.StatementNode{Child: trueCond}
			trueCond.SetParents(trueNode2)
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
			trueNode1.SetParents(root)
			falseNode1.SetParents(root)
			labels[root] = db.Must

			return labelTestCase{
				Name:   "nested-if-else-extra",
				Root:   root,
				Leaf:   leaf,
				Labels: labels,
				Logs:   make([]model.LogType, 0),
			}
		},
		func() labelTestCase {
			endIf := &db.EndConditionalNode{}
			root := &db.ConditionalNode{TrueChild: endIf, FalseChild: endIf}
			endIf.SetParents(root)
			labels := make(map[db.Node]db.ExecutionLabel)
			labels[root] = db.Must
			labels[endIf] = db.Must

			return labelTestCase{
				Name:   "dead-if-else",
				Root:   root,
				Leaf:   endIf,
				Labels: labels,
				Logs:   make([]model.LogType, 0),
			}
		},
		func() labelTestCase {
			end := &db.EndConditionalNode{}
			t1 := &db.FunctionNode{Child: end}
			f1 := &db.StatementNode{
				Filename:   "/some/path/to/file.go",
				LogRegex:   "this is a log message: .*",
				LineNumber: 67,
				Child:      end,
			}
			end.SetParents(t1)
			end.SetParents(f1)

			root := &db.ConditionalNode{TrueChild: t1, FalseChild: f1}
			t1.SetParents(root)
			f1.SetParents(root)
			labels := make(map[db.Node]db.ExecutionLabel)
			labels[end] = db.Must
			labels[t1] = db.MustNot //this is not handled yet
			labels[f1] = db.Must
			labels[root] = db.Must

			logs := []model.LogType{
				{
					LineNumber: 67,
					FilePath:   "/some/path/to/file.go",
					Regex:      "this is a log message: .*",
				},
			}

			return labelTestCase{
				Name:   "log-if-else",
				Root:   root,
				Leaf:   end,
				Labels: labels,
				Logs:   logs,
			}
		},
		func() labelTestCase {
			end := &db.EndConditionalNode{}
			t1 := &db.FunctionNode{Child: end}
			end.SetParents(t1)
			extraNode2 := &db.FunctionNode{LineNumber: 5, Child: end}
			end.SetParents(extraNode2)
			extraNode1 := &db.FunctionNode{LineNumber: 4, Child: extraNode2}
			extraNode2.SetParents(extraNode1)
			f1 := &db.StatementNode{
				Filename:   "/some/path/to/file.go",
				LogRegex:   "err: .*",
				LineNumber: 2,
				Child:      extraNode1,
			}
			extraNode1.SetParents(f1)

			root := &db.ConditionalNode{TrueChild: t1, FalseChild: f1}
			t1.SetParents(root)
			f1.SetParents(root)
			labels := make(map[db.Node]db.ExecutionLabel)
			labels[end] = db.Must
			labels[t1] = db.MustNot //this is not handled yet
			labels[f1] = db.Must
			labels[extraNode1] = db.Must
			labels[extraNode2] = db.Must
			labels[root] = db.Must

			logs := []model.LogType{
				{
					LineNumber: 2,
					FilePath:   "/some/path/to/file.go",
					Regex:      "err: .*",
				},
			}

			return labelTestCase{
				Name:   "log-if-else-ext",
				Root:   root,
				Leaf:   end,
				Labels: labels,
				Logs:   logs,
			}
		},
		func() labelTestCase {
			labels := make(map[db.Node]db.ExecutionLabel)

			end := &db.EndConditionalNode{}
			labels[end] = db.Must

			// outer false branch
			fEnd := &db.FunctionNode{Child: end, LineNumber: 8}
			fEndIf := &db.EndConditionalNode{Child: fEnd}
			fEnd.SetParents(fEndIf)
			ff := &db.StatementNode{
				LogRegex:   "i don't match",
				LineNumber: 7,
				Filename:   "somefile.go",
				Child:      fEndIf,
			}
			ft := &db.FunctionNode{Child: fEndIf, LineNumber: 6}
			fEndIf.SetParents(ft)
			fEndIf.SetParents(ff)
			fCond := &db.ConditionalNode{TrueChild: ft, FalseChild: ff, LineNumber: 5}
			ff.SetParents(fCond)
			ft.SetParents(fCond)

			labels[fEnd] = db.MustNot
			labels[fEndIf] = db.MustNot
			labels[ff] = db.MustNot
			labels[ft] = db.MustNot
			labels[fCond] = db.MustNot

			// outer true branch
			tEndIf := &db.EndConditionalNode{Child: end}
			tt := &db.FunctionNode{Child: tEndIf, LineNumber: 3}
			tExtraNode := &db.FunctionNode{Child: tEndIf, LineNumber: 4}
			tEndIf.SetParents(tt)
			tEndIf.SetParents(tExtraNode)
			tLog := &db.StatementNode{
				LogRegex:   "hello I am an .* error",
				LineNumber: 999,
				Filename:   "somefile.go",
				Child:      tExtraNode,
			}
			tExtraNode.SetParents(tLog)
			tCond := &db.ConditionalNode{
				TrueChild:  tt,
				FalseChild: tLog,
				LineNumber: 2,
			}
			tLog.SetParents(tCond)
			tt.SetParents(tCond)

			labels[tEndIf] = db.Must
			labels[tt] = db.MustNot
			labels[tExtraNode] = db.Must
			labels[tLog] = db.Must
			labels[tCond] = db.Must

			end.SetParents(tEndIf)
			end.SetParents(fEnd)

			root := &db.ConditionalNode{
				TrueChild:  tCond,
				FalseChild: fCond,
				LineNumber: 1,
			}
			tCond.SetParents(root)
			fCond.SetParents(root)
			labels[root] = db.Must

			logs := []model.LogType{
				{
					LineNumber: 999,
					FilePath:   "somefile.go",
					Regex:      "hello I am an .* error",
				},
			}

			return labelTestCase{
				Name:   "log-nested",
				Root:   root,
				Leaf:   end,
				Labels: labels,
				Logs:   logs,
			}
		},
	}

	for _, testCase := range cases {
		test := testCase()
		t.Run(test.Name, func(t *testing.T) {
			LabelParentNodes(test.Leaf, test.Logs)
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
