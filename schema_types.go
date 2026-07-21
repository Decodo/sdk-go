package decodo

// Schema defines the interface for request validation and target introspection.
type Schema interface {
	// GetRequestSchema returns a compiled schema validator for the given target, or nil.
	GetRequestSchema(target string) interface{}
	// ListTargets returns all available target identifiers.
	ListTargets() []string
	// GetTargetMeta returns metadata about a target.
	GetTargetMeta(target string) *TargetInfo
	// GetTargetParameterSchema returns the raw JSON schema for a target's parameters.
	GetTargetParameterSchema(target string) map[string]interface{}
	// GetSharedParameters returns shared parameter definitions from the IR.
	GetSharedParameters() map[string]interface{}
	// Version returns the IR version this schema was built from.
	Version() string
}

// TargetInfo contains metadata about a scrape target.
type TargetInfo struct {
	Group          string
	ResponseFormat string
	Parameters     []string
}

// ScrapeRequest is implemented by all target parameter types.
type ScrapeRequest interface {
	GetTarget() string
}
