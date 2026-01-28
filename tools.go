//go:build tools
// +build tools

// Package tools contains tool dependencies for code generation.
// This file is not compiled into the binary but ensures go mod tidy
// keeps the tool dependencies in go.mod.
package tools

import (
	_ "github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen"
)
