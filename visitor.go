package main

import (
  "go/ast"
  "go/token"
  "go/parser"
  "go/build"
  "path/filepath"
)

func FindFuncs(fp string) ([]*FuncDefinition, error) {
  dir, file := filepath.Split(fp)
  pkg, err := build.Import(dir, ".", build.FindOnly)
  if err != nil {
    return nil, err
  }

  absPath := filepath.Join(pkg.Dir, file)
  set := token.NewFileSet()
  parsedFile, err := parser.ParseFile(set, absPath, nil, 0)
  if err != nil {
    return nil, err
  }

  visitor := &visitor{
    set:      set,
    file: fp,
  }

  ast.Walk(visitor, parsedFile)
  return visitor.Functions, nil
}

type visitor struct {
  set    *token.FileSet
  file string
  Functions   []*FuncDefinition
  Structs []*StructDefinition
}

type FuncDefinition struct {
  Name      string
  File string
  Receiver string
  StartLine int
  StartCol  int
  EndLine   int
  EndCol    int
}

type StructDefinition struct {
  Name      string
}

func (v *visitor) Visit(node ast.Node) ast.Visitor {
  switch n := node.(type) {
  case *ast.FuncDecl:
    start := v.set.Position(node.Pos())
    end := v.set.Position(node.End())
    rec := "-"
    if n.Recv != nil {
      switch e := n.Recv.List[0].Type.(type) {
      case *ast.Ident:
        rec = e.Name
      case *ast.StarExpr:
        if id, ok := e.X.(*ast.Ident); ok {
          rec = id.Name
        }
      }
    }

    fe := &FuncDefinition{
      Name:      n.Name.Name,
      StartLine: start.Line,
      StartCol:  start.Column,
      EndLine:   end.Line,
      EndCol:    end.Column,
      File: v.file,
      Receiver: rec,
    }

    v.Functions = append(v.Functions, fe)
  }
  return v
}