package brain

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// Load reads the AI_BRAIN directory tree and returns a populated Brain.
// If the AI_BRAIN directory does not exist, it falls back to reading prompt.txt
// from basePath and sets b.FallbackPrompt accordingly.
func Load(basePath string) *Brain {
	b := &Brain{
		Core:              make(map[string]string),
		Flows:             make(map[string]string),
		Knowledge:         make(map[string]string),
		GoAngkasaInfo:     make(map[string]string),
		GoFlexiInfo:       make(map[string]string),
		Templates:         make(map[string]string),
		GoAngkasaProducts: make(map[string]map[string]ProductData),
		GoFlexiProducts:   make(map[string]map[string]ProductData),
	}

	brainDir := basePath

	info, err := os.Stat(brainDir)
	if err != nil || !info.IsDir() {
		log.Printf("[brain] AI_BRAIN directory not found at %s, falling back to prompt.txt", brainDir)
		b.loadFallback(basePath)
		return b
	}

	b.loadTextFiles(filepath.Join(brainDir, "core"), b.Core)
	b.loadTextFiles(filepath.Join(brainDir, "flows"), b.Flows)
	b.loadTextFiles(filepath.Join(brainDir, "knowledge", "general"), b.Knowledge)
	b.loadTextFiles(filepath.Join(brainDir, "knowledge", "goangkasa"), b.GoAngkasaInfo)
	b.loadTextFiles(filepath.Join(brainDir, "knowledge", "goflexi"), b.GoFlexiInfo)
	b.loadTextFiles(filepath.Join(brainDir, "templates"), b.Templates)
	b.loadMasterPrompt(filepath.Join(brainDir, "system", "master_prompt.txt"))

	b.loadProducts(
		filepath.Join(brainDir, "knowledge", "goangkasa", "products"),
		b.GoAngkasaProducts,
		"goangkasa",
	)
	b.loadProducts(
		filepath.Join(brainDir, "knowledge", "goflexi", "products"),
		b.GoFlexiProducts,
		"goflexi",
	)

	b.Loaded = true

	goAngkasaBrands := len(b.GoAngkasaProducts)
	goFlexiBrands := len(b.GoFlexiProducts)

	log.Printf("[brain] Loaded AI_BRAIN: %d core, %d flows, %d knowledge, %d goangkasa brands, %d goflexi brands, %d templates",
		len(b.Core), len(b.Flows), len(b.Knowledge), goAngkasaBrands, goFlexiBrands, len(b.Templates))

	return b
}

// Reload re-reads the AI_BRAIN directory into an existing Brain instance.
// It acquires a write lock during the reload and preserves thread safety.
func Reload(basePath string, b *Brain) error {
	fresh := Load(basePath)

	b.mu.Lock()
	defer b.mu.Unlock()

	b.Core = fresh.Core
	b.Flows = fresh.Flows
	b.Knowledge = fresh.Knowledge
	b.GoAngkasaInfo = fresh.GoAngkasaInfo
	b.GoFlexiInfo = fresh.GoFlexiInfo
	b.Templates = fresh.Templates
	b.GoAngkasaProducts = fresh.GoAngkasaProducts
	b.GoFlexiProducts = fresh.GoFlexiProducts
	b.MasterPrompt = fresh.MasterPrompt
	b.FallbackPrompt = fresh.FallbackPrompt
	b.Loaded = fresh.Loaded

	return nil
}

func (b *Brain) loadFallback(basePath string) {
	promptPath := filepath.Join(basePath, "prompt.txt")
	data, err := os.ReadFile(promptPath)
	if err != nil {
		log.Printf("[brain] prompt.txt not found at %s: %v", promptPath, err)
		b.FallbackPrompt = "You are a helpful assistant."
		return
	}
	b.FallbackPrompt = string(data)
	log.Printf("[brain] Using fallback prompt.txt (%d bytes)", len(data))
}

func (b *Brain) loadTextFiles(dir string, target map[string]string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("[brain] Error reading directory %s: %v", dir, err)
		}
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".txt") {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".txt")
		fullPath := filepath.Join(dir, entry.Name())

		data, err := os.ReadFile(fullPath)
		if err != nil {
			log.Printf("[brain] Error reading %s: %v", fullPath, err)
			continue
		}

		target[name] = string(data)
	}
}

func (b *Brain) loadMasterPrompt(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("[brain] Error reading master prompt %s: %v", path, err)
		}
		return
	}
	b.MasterPrompt = string(data)
}

func (b *Brain) loadProducts(productsDir string, target map[string]map[string]ProductData, source string) {
	brandEntries, err := os.ReadDir(productsDir)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("[brain] Error reading products directory %s: %v", productsDir, err)
		}
		return
	}

	for _, brandEntry := range brandEntries {
		if !brandEntry.IsDir() {
			continue
		}

		brandSlug := brandEntry.Name()
		brandDir := filepath.Join(productsDir, brandSlug)

		modelEntries, err := os.ReadDir(brandDir)
		if err != nil {
			log.Printf("[brain] Error reading brand directory %s: %v", brandDir, err)
			continue
		}

		for _, modelEntry := range modelEntries {
			if modelEntry.IsDir() || !strings.HasSuffix(modelEntry.Name(), ".json") {
				continue
			}

			modelSlug := strings.TrimSuffix(modelEntry.Name(), ".json")
			modelPath := filepath.Join(brandDir, modelEntry.Name())

			data, err := os.ReadFile(modelPath)
			if err != nil {
				log.Printf("[brain] Error reading product %s: %v", modelPath, err)
				continue
			}

			var pd ProductData
			if err := json.Unmarshal(data, &pd); err != nil {
				log.Printf("[brain] Error parsing product JSON %s: %v", modelPath, err)
				continue
			}

			pd.Source = source

			if target[brandSlug] == nil {
				target[brandSlug] = make(map[string]ProductData)
			}
			target[brandSlug][modelSlug] = pd
		}
	}
}
