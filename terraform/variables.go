// Copyright 2021 Nicolas Thauvin. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package terraform

import (
	"fmt"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"os"
)

type Variables struct {
	Variables []Variable `hcl:"variable,block"`
}

type Variable struct {
	Name        string   `hcl:"name,label"`
	Description string   `hcl:"description"`
	Data        hcl.Body `hcl:",remain"`
}

type VarString struct {
	Value string `hcl:"default"`
}

type VarMachines struct {
	Value map[string]Machine `hcl:"default"`
}

func ParseModuleVariables(path string) (Variables, error) {

	parser := hclparse.NewParser()
	f, diags := parser.ParseHCLFile(path)

	wr := hcl.NewDiagnosticTextWriter(
		os.Stderr,      // writer to send messages to
		parser.Files(), // the parser's file cache, for source snippets
		78,             // wrapping width
		true,           // generate colored/highlighted output
	)

	if diags.HasErrors() {
		wr.WriteDiagnostics(diags)
		return Variables{}, fmt.Errorf("could not parse HCL configuration")
	}

	var vars Variables

	moreDiags := gohcl.DecodeBody(f.Body, nil, &vars)
	diags = append(diags, moreDiags...)

	if diags.HasErrors() {
		wr.WriteDiagnostics(diags)
		return Variables{}, fmt.Errorf("could not parse HCL configuration")
	}

	return vars, nil
}

func ParseVariableDefault(v Variable, val interface{}) error {

	wr := hcl.NewDiagnosticTextWriter(
		os.Stderr, // writer to send messages to
		nil,       // the parser's file cache, for source snippets
		78,        // wrapping width
		true,      // generate colored/highlighted output
	)

	diags := gohcl.DecodeBody(v.Data, nil, val)
	if diags.HasErrors() {
		wr.WriteDiagnostics(diags)
		return fmt.Errorf("could not parse terraform variable defaut value for %s", v.Name)
	}

	return nil
}

func ShowModuleVariable(val interface{}) {

	f := hclwrite.NewEmptyFile()
	gohcl.EncodeIntoBody(val, f.Body())

	fmt.Printf("%s\n", f.Bytes())
}
