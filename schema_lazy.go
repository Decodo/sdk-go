package decodo

import (
	"fmt"
	"os"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// SharedDefaultSchema is the package-level default Schema used by NewClient when no Schema
// is provided. It auto-loads the remote IR on first use, caches locally for 24 hours, and
// silently disables validation if the fetch fails.
var SharedDefaultSchema Schema = &lazyRemoteSchema{
	opts: RemoteSchemaOptions{TTLMs: 24 * 60 * 60 * 1000},
}

type lazyRemoteSchema struct {
	once   sync.Once
	schema *RemoteSchema
	opts   RemoteSchemaOptions
}

func (s *lazyRemoteSchema) load() {
	s.once.Do(func() {
		schema, err := LoadRemoteSchema(s.opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "decodo: failed to load remote schema: %v (validation disabled for this session)\n", err)
			return
		}
		s.schema = schema
	})
}

func (s *lazyRemoteSchema) GetRequestSchema(target string) *jsonschema.Schema {
	s.load()
	if s.schema == nil {
		return nil
	}
	return s.schema.GetRequestSchema(target)
}

func (s *lazyRemoteSchema) ListTargets() []string {
	s.load()
	if s.schema == nil {
		return nil
	}
	return s.schema.ListTargets()
}

func (s *lazyRemoteSchema) GetTargetMeta(target string) *TargetInfo {
	s.load()
	if s.schema == nil {
		return nil
	}
	return s.schema.GetTargetMeta(target)
}

func (s *lazyRemoteSchema) GetTargetParameterSchema(target string) map[string]interface{} {
	s.load()
	if s.schema == nil {
		return nil
	}
	return s.schema.GetTargetParameterSchema(target)
}

func (s *lazyRemoteSchema) GetSharedParameters() map[string]interface{} {
	s.load()
	if s.schema == nil {
		return nil
	}
	return s.schema.GetSharedParameters()
}

func (s *lazyRemoteSchema) Version() string {
	s.load()
	if s.schema == nil {
		return ""
	}
	return s.schema.Version()
}
