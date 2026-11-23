// Copyright (C) 2026 The OpenEverest Contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/scanner"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// goAPIEntries returns the exported API of every Go package in a directory
// pattern such as contract/...: one entry per package, holding one line per
// exported declaration. A declaration is reduced to what a caller depends
// on, so documentation, layout, parameter names, function bodies and
// unexported identifiers do not count. The constants this command
// generates are left out, since they are derived from the fingerprint.
func goAPIEntries(root, pattern string) ([]entry, error) {
	dir, ok := strings.CutSuffix(pattern, "/...")
	if !ok {
		return nil, fmt.Errorf("pattern %q names no directory of Go packages", pattern)
	}
	var entries []entry
	err := filepath.WalkDir(filepath.Join(root, filepath.FromSlash(dir)), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if name := d.Name(); path != filepath.Join(root, filepath.FromSlash(dir)) &&
			(name == "testdata" || strings.HasPrefix(name, "_") || strings.HasPrefix(name, ".")) {
			return filepath.SkipDir
		}
		decls, err := packageAPI(path)
		if err != nil {
			return err
		}
		if decls == nil {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		entries = append(entries, entry{key: filepath.ToSlash(rel), data: []byte(strings.Join(decls, "\n"))})
		return nil
	})
	return entries, err
}

// packageAPI returns the exported declarations of the Go package in dir,
// sorted, or nil when dir holds no Go package.
func packageAPI(dir string) ([]string, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		return nil, err
	}
	var decls []string
	found := false
	for _, file := range files {
		name := filepath.Base(file)
		if strings.HasSuffix(name, "_test.go") || name == outputFile {
			continue
		}
		found = true
		src, err := os.ReadFile(file) //nolint:gosec // a Go file of this repository
		if err != nil {
			return nil, err
		}
		fileDecls, err := fileAPI(file, src)
		if err != nil {
			return nil, err
		}
		decls = append(decls, fileDecls...)
	}
	if !found {
		return nil, nil
	}
	slices.Sort(decls)
	return slices.Compact(decls), nil
}

// fileAPI returns the exported declarations of one Go file.
func fileAPI(name string, src []byte) ([]string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, name, src, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	var decls []string
	for _, d := range f.Decls {
		var nodes []ast.Node
		switch d := d.(type) {
		case *ast.FuncDecl:
			nodes = exportedFunc(d)
		case *ast.GenDecl:
			nodes = exportedGen(d)
		}
		for _, n := range nodes {
			s, err := canonicalGo(fset, n)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}
			decls = append(decls, s)
		}
	}
	return decls, nil
}

// exportedFunc returns the signature of an exported function, or of an
// exported method of an exported type.
func exportedFunc(d *ast.FuncDecl) []ast.Node {
	if !d.Name.IsExported() {
		return nil
	}
	if d.Recv != nil && (len(d.Recv.List) != 1 || !receiverTypeName(d.Recv.List[0].Type).IsExported()) {
		return nil
	}
	return []ast.Node{&ast.FuncDecl{
		Recv: unnamedFields(d.Recv),
		Name: d.Name,
		Type: &ast.FuncType{
			TypeParams: d.Type.TypeParams,
			Params:     unnamedFields(d.Type.Params),
			Results:    unnamedFields(d.Type.Results),
		},
	}}
}

// receiverTypeName returns the name of the type of a method receiver such
// as *T or T[K].
func receiverTypeName(expr ast.Expr) *ast.Ident {
	for {
		switch e := expr.(type) {
		case *ast.StarExpr:
			expr = e.X
		case *ast.ParenExpr:
			expr = e.X
		case *ast.IndexExpr:
			expr = e.X
		case *ast.IndexListExpr:
			expr = e.X
		case *ast.Ident:
			return e
		default:
			return ast.NewIdent("_")
		}
	}
}

// unnamedFields returns a parameter list with one unnamed field per
// parameter, so renaming a parameter does not change the signature.
func unnamedFields(fl *ast.FieldList) *ast.FieldList {
	if fl == nil {
		return nil
	}
	out := &ast.FieldList{}
	for _, f := range fl.List {
		n := max(len(f.Names), 1)
		for range n {
			out.List = append(out.List, &ast.Field{Type: f.Type})
		}
	}
	return out
}

// exportedGen returns the exported specs of a type, const or var
// declaration, each as a declaration of its own.
func exportedGen(d *ast.GenDecl) []ast.Node {
	var nodes []ast.Node
	// In a const group, a spec without values repeats the type and values
	// of the last one that has them.
	var lastType ast.Expr
	var lastValues []ast.Expr
	for i, s := range d.Specs {
		switch s := s.(type) {
		case *ast.TypeSpec:
			if s.Name.IsExported() {
				nodes = append(nodes, &ast.GenDecl{Tok: token.TYPE, Specs: []ast.Spec{&ast.TypeSpec{
					Name:       s.Name,
					TypeParams: s.TypeParams,
					Assign:     s.Assign,
					Type:       exportedType(s.Type),
				}}})
			}
		case *ast.ValueSpec:
			typ, values := s.Type, s.Values
			if d.Tok == token.CONST {
				if typ == nil && values == nil {
					typ, values = lastType, lastValues
				}
				lastType, lastValues = typ, values
			}
			for j, name := range s.Names {
				if !name.IsExported() {
					continue
				}
				spec := &ast.ValueSpec{Names: []*ast.Ident{name}, Type: typ}
				if j < len(values) {
					spec.Values = []ast.Expr{values[j]}
				}
				decl := &ast.GenDecl{Tok: d.Tok, Specs: []ast.Spec{spec}}
				if d.Tok == token.CONST && usesIota(spec.Values) {
					nodes = append(nodes, iotaDecl{GenDecl: decl, iota: i})
					continue
				}
				nodes = append(nodes, decl)
			}
		}
	}
	return nodes
}

// iotaDecl is a constant declaration whose value depends on its index in
// its group.
type iotaDecl struct {
	*ast.GenDecl

	iota int
}

func usesIota(values []ast.Expr) bool {
	found := false
	for _, v := range values {
		ast.Inspect(v, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok && id.Name == "iota" {
				found = true
			}
			return !found
		})
	}
	return found
}

// exportedType returns a type with the unexported fields of its structs
// removed. Embedded fields stay, since they promote exported fields and
// methods; so do the methods of interfaces, which decide who can
// implement them.
func exportedType(expr ast.Expr) ast.Expr {
	st, ok := expr.(*ast.StructType)
	if !ok {
		return expr
	}
	out := &ast.StructType{Fields: &ast.FieldList{}}
	for _, f := range st.Fields.List {
		field := &ast.Field{Type: exportedType(f.Type), Tag: f.Tag}
		if len(f.Names) > 0 {
			for _, n := range f.Names {
				if n.IsExported() {
					field.Names = append(field.Names, n)
				}
			}
			if len(field.Names) == 0 {
				continue
			}
		}
		out.Fields.List = append(out.Fields.List, field)
	}
	return out
}

// canonicalGo prints a declaration as its tokens separated by single
// spaces, so line breaks, comments and the quoting of strings do not
// count.
func canonicalGo(fset *token.FileSet, n ast.Node) (string, error) {
	var prefix string
	if d, ok := n.(iotaDecl); ok {
		prefix = "iota=" + strconv.Itoa(d.iota) + " "
		n = d.GenDecl
	}
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, n); err != nil {
		return "", err
	}
	toks, err := scanTokens(buf.Bytes())
	if err != nil {
		return "", err
	}
	words := make([]string, 0, len(toks))
	for i, t := range toks {
		// A line break before a closing bracket requires a trailing comma.
		if t.tok == token.COMMA && i+1 < len(toks) && closing(toks[i+1].tok) {
			continue
		}
		if t.lit != "" {
			words = append(words, t.lit)
		} else {
			words = append(words, t.tok.String())
		}
	}
	return prefix + strings.Join(words, " "), nil
}

type goToken struct {
	tok token.Token
	lit string
}

// scanTokens returns the tokens of Go source, without comments and
// semicolons, with every string literal quoted the same way.
func scanTokens(src []byte) ([]goToken, error) {
	var scanErr error
	onError := func(pos token.Position, msg string) { scanErr = fmt.Errorf("%s: %s", pos, msg) }
	var s scanner.Scanner
	s.Init(token.NewFileSet().AddFile("", -1, len(src)), src, onError, 0)
	var toks []goToken
	for {
		_, t, lit := s.Scan()
		if t == token.EOF {
			return toks, scanErr
		}
		if t == token.SEMICOLON {
			continue
		}
		if v, err := strconv.Unquote(lit); t == token.STRING && err == nil {
			lit = strconv.Quote(v)
		}
		toks = append(toks, goToken{t, lit})
	}
}

func closing(t token.Token) bool {
	return t == token.RPAREN || t == token.RBRACE || t == token.RBRACK
}
