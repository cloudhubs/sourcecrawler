package cfg

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/cfg"
)

//---- Branch Labels ----------
type ExecutionLabel int

const (
	NoLabel ExecutionLabel = iota
	Must
	May
	MustNot
)

func (s ExecutionLabel) String() string {
	return [...]string{"NoLabel", "Must", "May", "MustNot"}[s]
}

type Wrapper interface {
	AddParent(w Wrapper)
	RemoveParent(w Wrapper)
	GetParents() []Wrapper
	AddChild(w Wrapper)
	RemoveChild(w Wrapper)
	GetChildren() []Wrapper
	GetOuterWrapper() Wrapper
	SetOuterWrapper(w Wrapper)
	GetLabel() ExecutionLabel
	SetLabel(label ExecutionLabel)

	GetFileSet() *token.FileSet
	GetASTs() []*ast.File
	//GetPathList() PathList
}

type FnWrapper struct {
	Fn         ast.Node // *ast.FuncDel or *ast.FuncLit
	FirstBlock Wrapper
	Parents    []Wrapper
	Outer      Wrapper
	// ...?

	Label        ExecutionLabel
	Fset         *token.FileSet
	ASTs         []*ast.File
	ParamsToArgs map[*ast.Object]ast.Expr
	//PathList PathList
}

type BlockWrapper struct {
	Block   *cfg.Block
	Parents []Wrapper
	Succs   []Wrapper
	Outer   Wrapper
	Label   ExecutionLabel
	//PathList PathList
}

// ------------------ FnWrapper ----------------------

func (fn *FnWrapper) AddParent(w Wrapper) {
	if fn.Parents == nil {
		fn.Parents = make([]Wrapper, 0)
	}
	if w != nil {
		if w != nil {
			found := false
			for _, p := range fn.Parents {
				if p == w {
					found = true
					break
				}
			}
			if !found {
				fn.Parents = append(fn.Parents, w)
			}
		}
	}
}

func (fn *FnWrapper) RemoveParent(w Wrapper) {
	for i, p := range fn.Parents {
		if p == w {
			if i < len(fn.Parents)-1 {
				fn.Parents = append(fn.Parents[:i], fn.Parents[i+1:]...)
			} else {
				fn.Parents = fn.Parents[:i]
			}
		}
	}
}

func (fn *FnWrapper) GetParents() []Wrapper {
	return fn.Parents
}

func (fn *FnWrapper) AddChild(w Wrapper) {
	fn.FirstBlock = w
}

func (fn *FnWrapper) RemoveChild(w Wrapper) {
	fn.FirstBlock = nil
}

func (fn *FnWrapper) GetChildren() []Wrapper {
	if fn.FirstBlock == nil {
		return []Wrapper{}
	}
	return []Wrapper{fn.FirstBlock}
}

func (fn *FnWrapper) GetOuterWrapper() Wrapper {
	return fn.Outer
}

func (fn *FnWrapper) SetOuterWrapper(w Wrapper) {
	fn.Outer = w
}

//must always be defined by the outermost wrapper
func (fn *FnWrapper) GetFileSet() *token.FileSet {
	if fn.Fset != nil {
		return fn.Fset
	} else {
		if fn.Outer != nil {
			return fn.Outer.GetFileSet()
		}
		return nil
	}
}

//must always be defined by the outermost wrapper
func (fn *FnWrapper) GetASTs() []*ast.File {
	if fn.ASTs != nil {
		return fn.ASTs
	} else {
		if fn.Outer != nil {
			return fn.Outer.GetASTs()
		}
		return []*ast.File{}
	}
}

//Must always be defined by the outermost wrapper
//Assumptions: PathList has already been created
//func (fn *FnWrapper) GetPathList() PathList{
//	if fn.Outer != nil{
//		if fn == fn.Outer{
//			return fn.PathList
//		}
//		return fn.Outer.GetPathList()
//	}
//	return CreatePathList()
//}

func (fn *FnWrapper) GetLabel() ExecutionLabel {
	return fn.Label
}

func (fn *FnWrapper) SetLabel(label ExecutionLabel) {
	fn.Label = label
}

// ------------------ BlockWrapper ----------------------

func (b *BlockWrapper) AddParent(w Wrapper) {
	if b.Parents == nil {
		b.Parents = make([]Wrapper, 0)
	}
	if w != nil {
		found := false
		for _, p := range b.Parents {
			if p == w {
				found = true
				break
			}
		}
		if !found {
			b.Parents = append(b.Parents, w)
		}
	}
}

func (b *BlockWrapper) RemoveParent(w Wrapper) {
	for i, p := range b.Parents {
		if p == w {
			if i < len(b.Parents)-1 {
				b.Parents = append(b.Parents[:i], b.Parents[i+1:]...)
			} else {
				b.Parents = b.Parents[:i]
			}
		}
	}
}

func (b *BlockWrapper) GetParents() []Wrapper {
	return b.Parents
}

func (b *BlockWrapper) RemoveChild(w Wrapper) {
	for i, c := range b.Succs {
		if c == w {
			if i < len(b.Succs)-1 {
				b.Succs = append(b.Succs[:i], b.Succs[i+1:]...)
			} else {
				b.Succs = b.Succs[:i]
			}
		}
	}
}

func (b *BlockWrapper) AddChild(w Wrapper) {
	if w != nil {
		b.Succs = append(b.Succs, w)
	}
}

func (b *BlockWrapper) GetChildren() []Wrapper {
	return b.Succs
}

func (b *BlockWrapper) GetOuterWrapper() Wrapper {
	return b.Outer
}

func (b *BlockWrapper) SetOuterWrapper(w Wrapper) {
	b.Outer = w
}

func (b *BlockWrapper) GetFileSet() *token.FileSet {
	if b.Outer != nil {
		return b.Outer.GetFileSet()
	}
	return nil
}

func (b *BlockWrapper) GetASTs() []*ast.File {
	if b.Outer != nil {
		return b.Outer.GetASTs()
	}
	return []*ast.File{}
}

//Must always be defined by the outermost wrapper
//Assumptions: PathList has already been created
//func (b *BlockWrapper) GetPathList() PathList{
//	if b.Outer != nil{
//		if b == b.Outer{
//			return b.PathList
//		} else{
//			return b.Outer.GetPathList()
//		}
//	}
//
//	return CreatePathList()
//}

func (b *BlockWrapper) GetLabel() ExecutionLabel {
	return b.Label
}

func (b *BlockWrapper) SetLabel(label ExecutionLabel) {
	b.Label = label
}
