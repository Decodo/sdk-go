package decodo

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// BundledSchema uses the generated request schemas embedded in the binary.
type BundledSchema struct {
	once     sync.Once
	compiled map[string]*jsonschema.Schema
}

var sharedBundledSchema = &BundledSchema{}

// SharedBundledSchema is the default shared instance of BundledSchema.
var SharedBundledSchema Schema = sharedBundledSchema

func (s *BundledSchema) compile() {
	s.once.Do(func() {
		s.compiled = make(map[string]*jsonschema.Schema)
		compiler := jsonschema.NewCompiler()
		for target, rawSchema := range RequestSchemas {
			key := string(target)
			url := fmt.Sprintf("mem://%s.json", key)
			if err := compiler.AddResource(url, strings.NewReader(rawSchema)); err != nil {
				continue
			}
			compiled, err := compiler.Compile(url)
			if err != nil {
				continue
			}
			s.compiled[key] = compiled
		}
	})
}

func (s *BundledSchema) GetRequestSchema(target string) interface{} {
	s.compile()
	return s.compiled[target]
}

func (s *BundledSchema) ListTargets() []string {
	out := make([]string, 0, len(Targets))
	for _, t := range Targets {
		out = append(out, string(t))
	}
	return out
}

func (s *BundledSchema) GetTargetMeta(target string) *TargetInfo {
	m, ok := TargetMetaMap[Target(target)]
	if !ok {
		return nil
	}
	return &TargetInfo{
		Group:          m.Group,
		ResponseFormat: m.ResponseFormat,
		Parameters:     m.Parameters,
	}
}

func (s *BundledSchema) GetTargetParameterSchema(target string) map[string]interface{} {
	raw, ok := RequestSchemas[Target(target)]
	if !ok {
		return nil
	}
	var out map[string]interface{}
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}

func (s *BundledSchema) GetSharedParameters() map[string]interface{} {
	return map[string]interface{}{}
}

func (s *BundledSchema) Version() string {
	return ""
}
