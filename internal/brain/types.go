package brain

import "sync"

// ProductData represents a single product JSON file's content.
type ProductData struct {
	Brand     string                   `json:"brand"`
	BrandSlug string                   `json:"brandSlug"`
	Model     string                   `json:"model"`
	ModelSlug string                   `json:"modelSlug"`
	Category  string                   `json:"category,omitempty"`
	Source    string                   `json:"source,omitempty"`
	Variants  []map[string]interface{} `json:"variants"`
}

// ProductMatch holds a matched product with its source plan information.
type ProductMatch struct {
	Brand  string
	Model  string
	Source string // "goangkasa" or "goflexi"
	Data   ProductData
}

// Brain holds all AI_BRAIN content in memory with thread-safe access.
type Brain struct {
	mu                sync.RWMutex
	Core              map[string]string
	Flows             map[string]string
	Knowledge         map[string]string
	GoAngkasaInfo     map[string]string
	GoFlexiInfo       map[string]string
	Templates         map[string]string
	GoAngkasaProducts map[string]map[string]ProductData // brand -> model -> data
	GoFlexiProducts   map[string]map[string]ProductData // brand -> model -> data
	MasterPrompt      string
	FallbackPrompt    string
	Loaded            bool
}
