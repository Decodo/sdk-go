package main

import (
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type DataAPIIR struct {
	Label     string                       `json:"label"`
	BaseURL   string                       `json:"baseUrl"`
	Endpoints map[string]DataAPIEndpointIR `json:"endpoints"`
}

type DataAPIEndpointIR struct {
	Method          string      `json:"method"`
	Path            string      `json:"path"`
	QueryParameters *schemaNode `json:"queryParameters"`
	RequestSchema   *schemaNode `json:"requestSchema"`
	ResponseSchema  *schemaNode `json:"responseSchema"`
}

// schemaNode is the subset of JSON Schema the Data API section of the IR uses.
type schemaNode struct {
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Properties  map[string]*schemaNode `json:"properties"`
	Required    []string               `json:"required"`
	Items       *schemaNode            `json:"items"`
	Enum        []interface{}          `json:"enum"`
	OneOf       []*schemaNode          `json:"oneOf"`
	AnyOf       []*schemaNode          `json:"anyOf"`
}

type dataAPIEmitter struct {
	types   map[string]string
	request bool
}

func (n *schemaNode) isRequired(name string) bool {
	for _, r := range n.Required {
		if r == name {
			return true
		}
	}
	return false
}

// mergeVariants flattens an anyOf of object schemas into one object with every
// property optional, since a Go struct cannot express which variant was returned.
func mergeVariants(variants []*schemaNode) *schemaNode {
	merged := &schemaNode{Type: "object", Properties: map[string]*schemaNode{}}
	for _, v := range variants {
		for name, prop := range v.Properties {
			existing, ok := merged.Properties[name]
			if !ok {
				copied := *prop
				merged.Properties[name] = &copied
				continue
			}
			existing.Enum = append(existing.Enum, prop.Enum...)
		}
	}
	return merged
}

func isStringOrList(variants []*schemaNode) bool {
	if len(variants) != 2 {
		return false
	}
	var hasString, hasList bool
	for _, v := range variants {
		switch {
		case v.Type == "string":
			hasString = true
		case v.Type == "array" && v.Items != nil && v.Items.Type == "string":
			hasList = true
		}
	}
	return hasString && hasList
}

// resolve returns the node to emit for n, collapsing anyOf unions.
func resolve(n *schemaNode) *schemaNode {
	if len(n.AnyOf) > 0 {
		return mergeVariants(n.AnyOf)
	}
	return n
}

func (e *dataAPIEmitter) goType(n *schemaNode, name string) string {
	if len(n.OneOf) > 0 {
		if isStringOrList(n.OneOf) {
			return "[]string"
		}
		return "interface{}"
	}
	n = resolve(n)
	switch n.Type {
	case "string":
		return "string"
	case "integer", "number":
		// "number" is emitted as int to match the params generator.
		return "int"
	case "boolean":
		return "bool"
	case "array":
		if n.Items == nil {
			return "[]interface{}"
		}
		return "[]" + e.goType(n.Items, name+"Item")
	case "object":
		if len(n.Properties) == 0 {
			return "map[string]interface{}"
		}
		e.emitStruct(n, name)
		return name
	default:
		return "interface{}"
	}
}

func (e *dataAPIEmitter) emitStruct(n *schemaNode, name string) {
	if _, done := e.types[name]; done {
		return
	}
	e.types[name] = ""

	keys := make([]string, 0, len(n.Properties))
	for k := range n.Properties {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	fmt.Fprintf(&sb, "type %s struct {\n", name)
	for _, key := range keys {
		prop := n.Properties[key]
		field := toFieldName(key)
		goType := e.goType(prop, name+field)

		required := n.isRequired(key)
		scalar := goType == "string" || goType == "int" || goType == "bool"
		resolved := resolve(prop)
		structPtr := resolved.Type == "object" && len(resolved.Properties) > 0
		tag := key
		switch {
		case required:
		case e.request && scalar, structPtr:
			goType = "*" + goType
			tag += ",omitempty"
		default:
			tag += ",omitempty"
		}

		if prop.Description != "" {
			fmt.Fprintf(&sb, "\t// %s\n", strings.ReplaceAll(prop.Description, "\n", " "))
		}
		if len(prop.Enum) > 0 {
			vals := make([]string, 0, len(prop.Enum))
			for _, v := range prop.Enum {
				vals = append(vals, fmt.Sprintf("%v", v))
			}
			fmt.Fprintf(&sb, "\t// Valid values: %s\n", strings.Join(vals, ", "))
		}
		fmt.Fprintf(&sb, "\t%s %s `json:\"%s\"`\n", field, goType, tag)
	}
	sb.WriteString("}\n")
	e.types[name] = sb.String()
}

func (e *dataAPIEmitter) emitNamed(n *schemaNode, name string, request bool) {
	e.request = request
	n = resolve(n)
	if n.Type == "object" && len(n.Properties) > 0 {
		e.emitStruct(n, name)
		return
	}
	e.types[name] = fmt.Sprintf("type %s = %s\n", name, e.goType(n, name))
}

func generateDataAPIFile(ir *IR, outDir string) error {
	api := ir.APIs.DataAPI
	if len(api.Endpoints) == 0 {
		return fmt.Errorf("IR %s has no dataApi endpoints; data_api.go needs IR 2.3.0 or newer", ir.Version)
	}

	endpointKeys := make([]string, 0, len(api.Endpoints))
	for k := range api.Endpoints {
		endpointKeys = append(endpointKeys, k)
	}
	sort.Strings(endpointKeys)

	e := &dataAPIEmitter{types: map[string]string{}}
	for _, key := range endpointKeys {
		ep := api.Endpoints[key]
		base := toPascalCase(key)
		if ep.RequestSchema != nil {
			e.emitNamed(ep.RequestSchema, base+"Request", true)
		}
		if ep.QueryParameters != nil {
			e.emitNamed(ep.QueryParameters, base+"Query", true)
		}
		if ep.ResponseSchema != nil {
			e.emitNamed(ep.ResponseSchema, base+"Response", false)
		}
	}

	typeNames := make([]string, 0, len(e.types))
	for name := range e.types {
		typeNames = append(typeNames, name)
	}
	sort.Strings(typeNames)

	var sb strings.Builder
	sb.WriteString("// Code generated by cmd/codegen; DO NOT EDIT.\n\n")
	sb.WriteString("package decodo\n\n")

	sb.WriteString("// Data API routes. {id} is the crawl ID.\n")
	sb.WriteString("const (\n")
	for _, key := range endpointKeys {
		fmt.Fprintf(&sb, "\tdataAPIPath%s = %q\n", toPascalCase(key), api.Endpoints[key].Path)
	}
	sb.WriteString(")\n\n")

	for _, name := range typeNames {
		sb.WriteString(e.types[name])
		sb.WriteString("\n")
	}

	src, err := format.Source([]byte(sb.String()))
	if err != nil {
		return fmt.Errorf("formatting generated_data_api.go: %w", err)
	}

	outPath := filepath.Join(outDir, "generated_data_api.go")
	if err := os.WriteFile(outPath, src, 0644); err != nil {
		return fmt.Errorf("writing generated_data_api.go: %w", err)
	}
	fmt.Printf("Generated %s (%d endpoints, %d types)\n", outPath, len(endpointKeys), len(typeNames))
	return nil
}
