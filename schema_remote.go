package decodo

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// RemoteSchemaOptions configures how the remote schema is loaded.
type RemoteSchemaOptions struct {
	// URL overrides the auto-resolved IR URL.
	URL string
	// CachePath overrides the default cache file path (default: ~/.decodo/decodo.ir.json).
	CachePath string
	// TTLMs is the cache TTL in milliseconds. 0 means no expiry.
	TTLMs int64
}

type cachedIR struct {
	FetchedAt   int64    `json:"fetchedAt"`
	ResolvedURL string   `json:"resolvedUrl"`
	IR          remoteIR `json:"ir"`
}

type remoteIR struct {
	Version string       `json:"version"`
	APIs    remoteIRAPIs `json:"apis"`
}

type remoteIRAPIs struct {
	WebScrapingAPI webScrapingIR `json:"webScrapingApi"`
}

type webScrapingIR struct {
	Parameters map[string]interface{} `json:"parameters"`
	Targets    map[string]irTarget    `json:"targets"`
}

type irTarget struct {
	Group           string                 `json:"group"`
	ResponseFormat  string                 `json:"response_format"`
	ParameterSchema map[string]interface{} `json:"parameter_schema"`
}

// RemoteSchema fetches the IR from GCS and validates against the remote schema.
type RemoteSchema struct {
	mu               sync.RWMutex
	version          string
	compiled         map[string]*jsonschema.Schema
	targetKeys       []string
	targetMeta       map[string]*TargetInfo
	paramSchemas     map[string]map[string]interface{}
	sharedParameters map[string]interface{}
}

// LoadRemoteSchema loads the schema from GCS (or cache).
func LoadRemoteSchema(opts RemoteSchemaOptions) (*RemoteSchema, error) {
	ir, err := loadRemoteIR(opts)
	if err != nil {
		return nil, err
	}
	return buildRemoteSchema(ir)
}

func loadRemoteIR(opts RemoteSchemaOptions) (*remoteIR, error) {
	cachePath := expandPath(defaultIRCachePath)
	if opts.CachePath != "" {
		cachePath = expandPath(opts.CachePath)
	}

	var resolvedURL string
	var resolvedVersion string

	if opts.URL != "" {
		resolvedURL = opts.URL
	} else {
		loc, err := resolveLatestIR()
		if err == nil {
			resolvedURL = loc.URL
			resolvedVersion = loc.Version
		}
	}

	cached := readCachedIR(cachePath)

	if cached != nil && cached.ResolvedURL == resolvedURL {
		if opts.TTLMs == 0 || time.Now().UnixMilli()-cached.FetchedAt < opts.TTLMs {
			return &cached.IR, nil
		}
		if opts.URL != "" || cached.IR.Version == resolvedVersion {
			return &cached.IR, nil
		}
	}

	if resolvedURL == "" {
		if cached != nil {
			return &cached.IR, nil
		}
		return nil, fmt.Errorf("could not resolve IR URL and no cache available")
	}

	ir, err := fetchRemoteIR(resolvedURL)
	if err != nil {
		if cached != nil {
			return &cached.IR, nil
		}
		return nil, err
	}

	writeCachedIR(cachePath, &cachedIR{
		FetchedAt:   time.Now().UnixMilli(),
		ResolvedURL: resolvedURL,
		IR:          *ir,
	})

	return ir, nil
}

func readCachedIR(path string) *cachedIR {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var cached cachedIR
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil
	}
	return &cached
}

func writeCachedIR(path string, cached *cachedIR) {
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	data, _ := json.MarshalIndent(cached, "", "  ")
	_ = os.WriteFile(path, data, 0644)
}

func fetchRemoteIR(url string) (*remoteIR, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching IR from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("fetching IR from %s: HTTP %d", url, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading IR response: %w", err)
	}

	var ir remoteIR
	if err := json.Unmarshal(body, &ir); err != nil {
		return nil, fmt.Errorf("decoding IR: %w", err)
	}
	return &ir, nil
}

func buildRemoteSchema(ir *remoteIR) (*RemoteSchema, error) {
	s := &RemoteSchema{
		version:          ir.Version,
		compiled:         make(map[string]*jsonschema.Schema),
		targetMeta:       make(map[string]*TargetInfo),
		paramSchemas:     make(map[string]map[string]interface{}),
		sharedParameters: ir.APIs.WebScrapingAPI.Parameters,
	}

	compiler := jsonschema.NewCompiler()

	for key, target := range ir.APIs.WebScrapingAPI.Targets {
		s.targetKeys = append(s.targetKeys, key)

		// Build TargetInfo
		params := []string{}
		if props, ok := target.ParameterSchema["properties"].(map[string]interface{}); ok {
			for p := range props {
				if p != "target" {
					params = append(params, p)
				}
			}
		}
		s.targetMeta[key] = &TargetInfo{
			Group:          target.Group,
			ResponseFormat: target.ResponseFormat,
			Parameters:     params,
		}
		s.paramSchemas[key] = target.ParameterSchema

		// Compile JSON schema
		schemaJSON, err := json.Marshal(target.ParameterSchema)
		if err != nil {
			continue
		}
		schemaURL := fmt.Sprintf("mem://%s.json", key)
		if err := compiler.AddResource(schemaURL, strings.NewReader(string(schemaJSON))); err != nil {
			continue
		}
		compiled, err := compiler.Compile(schemaURL)
		if err != nil {
			continue
		}
		s.compiled[key] = compiled
	}

	return s, nil
}

func (s *RemoteSchema) GetRequestSchema(target string) interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.compiled[target]
}

func (s *RemoteSchema) ListTargets() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.targetKeys
}

func (s *RemoteSchema) GetTargetMeta(target string) *TargetInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.targetMeta[target]
}

func (s *RemoteSchema) GetTargetParameterSchema(target string) map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.paramSchemas[target]
}

func (s *RemoteSchema) GetSharedParameters() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sharedParameters
}

func (s *RemoteSchema) Version() string {
	return s.version
}
