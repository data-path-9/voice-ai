// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_infra

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestArchitecture_NoTypeAliases(t *testing.T) {
	forEachSIPGoFile(t, func(path string) {
		fileSet, file := parseSIPGoFile(t, path)
		for _, declaration := range file.Decls {
			generalDeclaration, ok := declaration.(*ast.GenDecl)
			if !ok || generalDeclaration.Tok != token.TYPE {
				continue
			}
			for _, spec := range generalDeclaration.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if ok && typeSpec.Assign.IsValid() {
					t.Fatalf("type alias is not allowed in SIP package: %s:%d", path, fileSet.Position(typeSpec.Assign).Line)
				}
			}
		}
	})
}

func TestArchitecture_CoreIsOnlyImportedByFacade(t *testing.T) {
	forEachSIPGoFile(t, func(path string) {
		relativePath := sipRelativePath(t, path)
		if strings.HasPrefix(relativePath, "infra/") || strings.HasPrefix(relativePath, "internal/core/") {
			return
		}

		_, file := parseSIPGoFile(t, path)
		for _, importSpec := range file.Imports {
			importPath, err := strconv.Unquote(importSpec.Path.Value)
			if err != nil {
				t.Fatalf("invalid import path in %s: %v", path, err)
			}
			if strings.HasSuffix(importPath, "/sip/internal/core") {
				t.Fatalf("only sip/infra may expose internal/core: %s imports %s", path, importPath)
			}
		}
	})
}

func forEachSIPGoFile(t *testing.T, visit func(path string)) {
	t.Helper()
	root := sipRoot(t)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".go") {
			visit(path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk SIP package: %v", err)
	}
}

func parseSIPGoFile(t *testing.T, path string) (*token.FileSet, *ast.File) {
	t.Helper()
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, path, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return fileSet, file
}

func sipRoot(t *testing.T) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller unavailable")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), ".."))
}

func sipRelativePath(t *testing.T, path string) string {
	t.Helper()
	relativePath, err := filepath.Rel(sipRoot(t), path)
	if err != nil {
		t.Fatalf("relative path for %s: %v", path, err)
	}
	return filepath.ToSlash(relativePath)
}
