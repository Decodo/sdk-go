package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

// IR types
type IR struct {
	Version string `json:"version"`
	APIs    IRAPIs `json:"apis"`
}

type IRAPIs struct {
	WebScrapingAPI WebScrapingAPIIR `json:"webScrapingApi"`
}

type WebScrapingAPIIR struct {
	Parameters map[string]json.RawMessage `json:"parameters"`
	Targets    map[string]IRTarget        `json:"targets"`
}

type IRTarget struct {
	Group           string          `json:"group"`
	ResponseFormat  string          `json:"response_format"`
	ParameterSchema json.RawMessage `json:"parameter_schema"`
}

type parameterSchema struct {
	Type                 string                    `json:"type"`
	Properties           map[string]propertySchema `json:"properties"`
	Required             []string                  `json:"required"`
	AdditionalProperties bool                      `json:"additionalProperties"`
}

type propertySchema struct {
	Type      string        `json:"type"`
	Const     string        `json:"const"`
	Enum      []interface{} `json:"enum"`
	MaxLength *int          `json:"maxLength"`
	Minimum   *float64      `json:"minimum"`
	Maximum   *float64      `json:"maximum"`
	Items     *itemsSchema  `json:"items"`
}

type itemsSchema struct {
	Type string `json:"type"`
}

// projectRoot returns the root of the Go module.
// When run via `go run ./cmd/codegen` from the project root, cwd IS the project root.
func projectRoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		panic(fmt.Sprintf("cannot determine working directory: %v", err))
	}
	return cwd
}

// goAcronyms is a set of well-known abbreviations that should be uppercased in Go identifiers.
var goAcronyms = map[string]string{
	"url":  "URL",
	"id":   "ID",
	"api":  "API",
	"http": "HTTP",
	"xhr":  "XHR",
	"html": "HTML",
	"json": "JSON",
	"xml":  "XML",
	"sql":  "SQL",
	"cdn":  "CDN",
	"ip":   "IP",
	"css":  "CSS",
	"uri":  "URI",
}

func toPascalCase(s string) string {
	parts := regexp.MustCompile(`[_\-\s]+`).Split(s, -1)
	var result strings.Builder
	for _, part := range parts {
		if len(part) == 0 {
			continue
		}
		lower := strings.ToLower(part)
		if acronym, ok := goAcronyms[lower]; ok {
			result.WriteString(acronym)
			continue
		}
		runes := []rune(part)
		result.WriteRune(unicode.ToUpper(runes[0]))
		result.WriteString(string(runes[1:]))
	}
	return result.String()
}

func toTargetConstName(targetKey string) string {
	pascal := toPascalCase(targetKey)
	if len(pascal) > 0 && unicode.IsDigit(rune(pascal[0])) {
		pascal = "_" + pascal
	}
	return "Target" + pascal
}

func toParamsTypeName(targetKey string) string {
	pascal := toPascalCase(targetKey)
	if len(pascal) > 0 && unicode.IsDigit(rune(pascal[0])) {
		pascal = "_" + pascal
	}
	return pascal + "Params"
}

func toBatchParamsTypeName(targetKey string) string {
	pascal := toPascalCase(targetKey)
	if len(pascal) > 0 && unicode.IsDigit(rune(pascal[0])) {
		pascal = "_" + pascal
	}
	return pascal + "BatchParams"
}

func toFieldName(jsonKey string) string {
	// Handle keys starting with digits - use "F" prefix to make them exported
	if len(jsonKey) > 0 && unicode.IsDigit(rune(jsonKey[0])) {
		return "F" + toPascalCase(jsonKey)
	}
	return toPascalCase(jsonKey)
}

func jsonSchemaTypeToGo(prop propertySchema) string {
	if prop.Const != "" {
		return "Target"
	}
	if prop.Type == "array" {
		if prop.Items != nil {
			switch prop.Items.Type {
			case "string":
				return "[]string"
			case "integer", "number":
				return "[]float64"
			default:
				return "[]interface{}"
			}
		}
		return "[]interface{}"
	}
	switch prop.Type {
	case "string":
		return "*string"
	case "boolean":
		return "*bool"
	case "integer", "number":
		return "*int"
	case "object":
		return "map[string]interface{}"
	default:
		return "interface{}"
	}
}

func isOptional(t string) bool {
	return strings.HasPrefix(t, "*") || strings.HasPrefix(t, "[]") || t == "interface{}" || strings.HasPrefix(t, "map[")
}

func fetchIR() (*IR, error) {
	root := projectRoot()

	// If DECODO_LOCAL_IR is set, skip GCS and use local file
	if os.Getenv("DECODO_LOCAL_IR") != "" {
		fmt.Println("DECODO_LOCAL_IR set, using local IR...")
		return loadLocalIR(root)
	}

	// Fetch from GCS first (always get latest)
	fmt.Println("Fetching IR from GCS...")
	listURL := "https://storage.googleapis.com/storage/v1/b/decodo-sdk-config/o?prefix=decodo-ir-v"
	resp, err := http.Get(listURL)
	if err != nil {
		// Fall back to local cache
		return loadLocalIR(root)
	}
	defer resp.Body.Close()

	type gcsItem struct {
		Name string `json:"name"`
	}
	type gcsList struct {
		Items []gcsItem `json:"items"`
	}
	var list gcsList
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return loadLocalIR(root)
	}

	re := regexp.MustCompile(`^decodo-ir-v(.+)\.json$`)
	var versions []string
	for _, item := range list.Items {
		m := re.FindStringSubmatch(item.Name)
		if m != nil {
			versions = append(versions, m[1])
		}
	}
	if len(versions) == 0 {
		return nil, fmt.Errorf("no IR versions found")
	}

	sort.Strings(versions)
	latest := versions[len(versions)-1]

	irURL := fmt.Sprintf("https://storage.googleapis.com/decodo-sdk-config/decodo-ir-v%s.json", latest)
	irResp, err := http.Get(irURL)
	if err != nil {
		return loadLocalIR(root)
	}
	defer irResp.Body.Close()

	data, err := io.ReadAll(irResp.Body)
	if err != nil {
		return loadLocalIR(root)
	}

	// Cache to inputs/ (gitignored, for local development)
	inputsDir := filepath.Join(root, "inputs")
	_ = os.MkdirAll(inputsDir, 0755)
	_ = os.WriteFile(filepath.Join(inputsDir, "decodo.ir.json"), data, 0644)

	var ir IR
	if err := json.Unmarshal(data, &ir); err != nil {
		return nil, fmt.Errorf("decoding IR: %w", err)
	}
	fmt.Printf("Fetched IR version %s\n", ir.Version)
	return &ir, nil
}

func loadLocalIR(root string) (*IR, error) {
	candidates := []string{
		filepath.Join(root, "inputs", "decodo.ir.json"),
		"/tmp/decodo.ir.json",
	}
	for _, localPath := range candidates {
		if data, err := os.ReadFile(localPath); err == nil {
			var ir IR
			if err := json.Unmarshal(data, &ir); err == nil {
				fmt.Printf("Using cached IR from %s (version %s)\n", localPath, ir.Version)
				return &ir, nil
			}
		}
	}
	return nil, fmt.Errorf("no local IR cache found")
}

// generateTargetsEnumFile writes the committed targets.go with just Target type + constants + Targets slice.
func generateTargetsEnumFile(ir *IR, outDir string) error {
	api := ir.APIs.WebScrapingAPI

	targetKeys := make([]string, 0, len(api.Targets))
	for k := range api.Targets {
		targetKeys = append(targetKeys, k)
	}
	sort.Strings(targetKeys)

	var sb strings.Builder

	sb.WriteString("// Code generated by cmd/codegen; DO NOT EDIT.\n")
	sb.WriteString("// Re-run `go run ./cmd/codegen` after updating the IR schema to regenerate.\n\n")
	sb.WriteString("package decodo\n\n")

	sb.WriteString("// Target represents the scrape target discriminator.\n")
	sb.WriteString("type Target string\n\n")
	sb.WriteString("// Target constants for all supported scrape targets.\n")
	sb.WriteString("const (\n")
	for _, key := range targetKeys {
		constName := toTargetConstName(key)
		sb.WriteString(fmt.Sprintf("\t%s Target = %q\n", constName, key))
	}
	sb.WriteString(")\n\n")

	sb.WriteString("// Targets contains all available scrape targets.\n")
	sb.WriteString("var Targets = []Target{\n")
	for _, key := range targetKeys {
		sb.WriteString(fmt.Sprintf("\t%s,\n", toTargetConstName(key)))
	}
	sb.WriteString("}\n")

	outPath := filepath.Join(outDir, "targets.go")
	if err := os.WriteFile(outPath, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("writing targets.go: %w", err)
	}
	fmt.Printf("Generated %s (%d targets)\n", outPath, len(targetKeys))
	return nil
}

// generateParamsFile writes the gitignored generated_params.go with all Params structs and TargetMetaMap.
func generateParamsFile(ir *IR, outDir string) error {
	api := ir.APIs.WebScrapingAPI

	targetKeys := make([]string, 0, len(api.Targets))
	for k := range api.Targets {
		targetKeys = append(targetKeys, k)
	}
	sort.Strings(targetKeys)

	var sb strings.Builder

	sb.WriteString("// Code generated by cmd/codegen; DO NOT EDIT.\n\n")
	sb.WriteString("package decodo\n\n")

	// Generate param structs
	for _, key := range targetKeys {
		target := api.Targets[key]
		typeName := toParamsTypeName(key)
		constName := toTargetConstName(key)

		var ps parameterSchema
		if err := json.Unmarshal(target.ParameterSchema, &ps); err != nil {
			continue
		}

		propKeys := make([]string, 0, len(ps.Properties))
		for k := range ps.Properties {
			if k != "target" {
				propKeys = append(propKeys, k)
			}
		}
		sort.Strings(propKeys)

		sb.WriteString(fmt.Sprintf("// %s contains parameters for the %s target.\n", typeName, key))
		sb.WriteString(fmt.Sprintf("type %s struct {\n", typeName))
		sb.WriteString("\tTarget Target `json:\"target\"`\n")

		for _, propKey := range propKeys {
			prop := ps.Properties[propKey]
			goType := jsonSchemaTypeToGo(prop)
			fn := toFieldName(propKey)
			jsonTag := propKey
			omitempty := ",omitempty"
			if !isOptional(goType) {
				omitempty = ""
			}

			if len(prop.Enum) > 0 {
				enumVals := make([]string, 0, len(prop.Enum))
				for _, e := range prop.Enum {
					enumVals = append(enumVals, fmt.Sprintf("%v", e))
				}
				sb.WriteString(fmt.Sprintf("\t// %s valid values: %s\n", fn, strings.Join(enumVals, ", ")))
			}

			sb.WriteString(fmt.Sprintf("\t%s %s `json:\"%s%s\"`\n", fn, goType, jsonTag, omitempty))
		}

		sb.WriteString("}\n\n")

		sb.WriteString("// GetTarget implements ScrapeRequest.\n")
		sb.WriteString(fmt.Sprintf("func (p *%s) GetTarget() string { return string(p.Target) }\n\n", typeName))

		sb.WriteString(fmt.Sprintf("// New%s creates a new %s with the target pre-set.\n", typeName, typeName))
		sb.WriteString(fmt.Sprintf("func New%s() *%s {\n", typeName, typeName))
		sb.WriteString(fmt.Sprintf("\treturn &%s{Target: %s}\n", typeName, constName))
		sb.WriteString("}\n\n")
	}

	// Generate batch param structs (url/query become []string)
	for _, key := range targetKeys {
		target := api.Targets[key]
		batchTypeName := toBatchParamsTypeName(key)
		constName := toTargetConstName(key)

		var ps parameterSchema
		if err := json.Unmarshal(target.ParameterSchema, &ps); err != nil {
			continue
		}

		propKeys := make([]string, 0, len(ps.Properties))
		for k := range ps.Properties {
			if k != "target" {
				propKeys = append(propKeys, k)
			}
		}
		sort.Strings(propKeys)

		sb.WriteString(fmt.Sprintf("// %s contains parameters for the %s target (batch mode).\n", batchTypeName, key))
		sb.WriteString(fmt.Sprintf("type %s struct {\n", batchTypeName))
		sb.WriteString("\tTarget Target `json:\"target\"`\n")

		for _, propKey := range propKeys {
			prop := ps.Properties[propKey]
			var goType string
			if propKey == "url" || propKey == "query" {
				goType = "[]string"
			} else {
				goType = jsonSchemaTypeToGo(prop)
			}
			fn := toFieldName(propKey)
			jsonTag := propKey
			omitempty := ",omitempty"
			if !isOptional(goType) {
				omitempty = ""
			}

			if len(prop.Enum) > 0 {
				enumVals := make([]string, 0, len(prop.Enum))
				for _, e := range prop.Enum {
					enumVals = append(enumVals, fmt.Sprintf("%v", e))
				}
				sb.WriteString(fmt.Sprintf("\t// %s valid values: %s\n", fn, strings.Join(enumVals, ", ")))
			}

			sb.WriteString(fmt.Sprintf("\t%s %s `json:\"%s%s\"`\n", fn, goType, jsonTag, omitempty))
		}

		sb.WriteString("}\n\n")

		sb.WriteString("// GetTarget implements ScrapeRequest.\n")
		sb.WriteString(fmt.Sprintf("func (p *%s) GetTarget() string { return string(p.Target) }\n\n", batchTypeName))

		sb.WriteString(fmt.Sprintf("// New%s creates a new %s with the target pre-set.\n", batchTypeName, batchTypeName))
		sb.WriteString(fmt.Sprintf("func New%s() *%s {\n", batchTypeName, batchTypeName))
		sb.WriteString(fmt.Sprintf("\treturn &%s{Target: %s}\n", batchTypeName, constName))
		sb.WriteString("}\n\n")
	}

	// TargetMetaMap
	sb.WriteString("// TargetMetaMap contains metadata for all targets.\n")
	sb.WriteString("var TargetMetaMap = map[Target]*TargetInfo{\n")
	for _, key := range targetKeys {
		target := api.Targets[key]
		constName := toTargetConstName(key)

		var ps parameterSchema
		_ = json.Unmarshal(target.ParameterSchema, &ps)

		sortedPropKeys := make([]string, 0, len(ps.Properties))
		for k := range ps.Properties {
			if k != "target" {
				sortedPropKeys = append(sortedPropKeys, k)
			}
		}
		sort.Strings(sortedPropKeys)

		params := make([]string, 0, len(sortedPropKeys))
		for _, propKey := range sortedPropKeys {
			params = append(params, fmt.Sprintf("%q", propKey))
		}

		sb.WriteString(fmt.Sprintf("\t%s: {\n", constName))
		sb.WriteString(fmt.Sprintf("\t\tGroup:          %q,\n", target.Group))
		sb.WriteString(fmt.Sprintf("\t\tResponseFormat: %q,\n", target.ResponseFormat))
		sb.WriteString(fmt.Sprintf("\t\tParameters:     []string{%s},\n", strings.Join(params, ", ")))
		sb.WriteString("\t},\n")
	}
	sb.WriteString("}\n")

	outPath := filepath.Join(outDir, "generated_params.go")
	if err := os.WriteFile(outPath, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("writing generated_params.go: %w", err)
	}
	fmt.Printf("Generated %s (%d targets)\n", outPath, len(targetKeys))
	return nil
}

func generateParametersFile(ir *IR, outDir string) error {
	api := ir.APIs.WebScrapingAPI

	type paramMeta struct {
		Type      string
		MaxLength *int
		Minimum   *float64
		Maximum   *float64
		Enum      []interface{}
		ItemsType string
	}

	collected := map[string]paramMeta{}
	var paramOrder []string

	targetKeys := make([]string, 0, len(api.Targets))
	for k := range api.Targets {
		targetKeys = append(targetKeys, k)
	}
	sort.Strings(targetKeys)

	for _, tKey := range targetKeys {
		target := api.Targets[tKey]
		var ps parameterSchema
		if err := json.Unmarshal(target.ParameterSchema, &ps); err != nil {
			continue
		}
		propKeys := make([]string, 0, len(ps.Properties))
		for k := range ps.Properties {
			propKeys = append(propKeys, k)
		}
		sort.Strings(propKeys)
		for _, pKey := range propKeys {
			if pKey == "target" {
				continue
			}
			if _, exists := collected[pKey]; exists {
				continue
			}
			paramOrder = append(paramOrder, pKey)
			prop := ps.Properties[pKey]
			m := paramMeta{Type: prop.Type}
			if prop.MaxLength != nil {
				m.MaxLength = prop.MaxLength
			}
			if prop.Minimum != nil {
				m.Minimum = prop.Minimum
			}
			if prop.Maximum != nil {
				m.Maximum = prop.Maximum
			}
			if len(prop.Enum) > 0 {
				m.Enum = prop.Enum
			}
			if prop.Items != nil {
				m.ItemsType = prop.Items.Type
			}
			collected[pKey] = m
		}
	}

	var sb strings.Builder
	sb.WriteString("// Code generated by cmd/codegen; DO NOT EDIT.\n\n")
	sb.WriteString("package decodo\n\n")
	sb.WriteString("// ParameterMeta contains metadata about a scrape parameter.\n")
	sb.WriteString("type ParameterMeta struct {\n")
	sb.WriteString("\tType      string\n")
	sb.WriteString("\tMaxLength int\n")
	sb.WriteString("\tMinimum   float64\n")
	sb.WriteString("\tMaximum   float64\n")
	sb.WriteString("\tEnum      []interface{}\n")
	sb.WriteString("\tItems     *ItemsMeta\n")
	sb.WriteString("}\n\n")
	sb.WriteString("// ItemsMeta contains metadata about array item types.\n")
	sb.WriteString("type ItemsMeta struct {\n")
	sb.WriteString("\tType string\n")
	sb.WriteString("}\n\n")
	sb.WriteString("// ParameterMetaMap contains metadata for all known parameters.\n")
	sb.WriteString("var ParameterMetaMap = map[string]ParameterMeta{\n")

	for _, pKey := range paramOrder {
		m := collected[pKey]
		parts := []string{fmt.Sprintf("Type: %q", m.Type)}
		if m.MaxLength != nil {
			parts = append(parts, fmt.Sprintf("MaxLength: %d", *m.MaxLength))
		}
		if m.Minimum != nil {
			parts = append(parts, fmt.Sprintf("Minimum: %g", *m.Minimum))
		}
		if m.Maximum != nil {
			parts = append(parts, fmt.Sprintf("Maximum: %g", *m.Maximum))
		}
		if len(m.Enum) > 0 {
			enumStrs := make([]string, 0, len(m.Enum))
			for _, e := range m.Enum {
				enumStrs = append(enumStrs, fmt.Sprintf("%q", fmt.Sprintf("%v", e)))
			}
			parts = append(parts, fmt.Sprintf("Enum: []interface{}{%s}", strings.Join(enumStrs, ", ")))
		}
		if m.ItemsType != "" {
			parts = append(parts, fmt.Sprintf("Items: &ItemsMeta{Type: %q}", m.ItemsType))
		}
		sb.WriteString(fmt.Sprintf("\t%q: {%s},\n", pKey, strings.Join(parts, ", ")))
	}

	sb.WriteString("}\n")

	outPath := filepath.Join(outDir, "generated_parameters.go")
	if err := os.WriteFile(outPath, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("writing generated_parameters.go: %w", err)
	}
	fmt.Printf("Generated %s (%d parameters)\n", outPath, len(paramOrder))
	return nil
}

func main() {
	ir, err := fetchIR()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading IR: %v\n", err)
		os.Exit(1)
	}

	outDir := projectRoot()

	if err := generateTargetsEnumFile(ir, outDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating targets enum: %v\n", err)
		os.Exit(1)
	}

	if err := generateParamsFile(ir, outDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating params: %v\n", err)
		os.Exit(1)
	}

	if err := generateParametersFile(ir, outDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating parameters: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nCodegen complete! IR version: %s\n", ir.Version)
}
