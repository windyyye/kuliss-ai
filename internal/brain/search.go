package brain

import (
	"fmt"
	"sort"
	"strings"
)

var knownBrands = []string{
	"apple", "samsung", "google", "honor", "oppo", "xiaomi",
	"vivo", "realme", "asus", "iqoo", "tecno", "oneplus",
	"nothing", "red magic", "infinix",
}

// SearchProducts searches both GoAngkasa and GoFlexi product catalogs
// for matches against the user query. It returns up to maxResults matches,
// prioritizing brand+model matches over model-only matches.
func (b *Brain) SearchProducts(query string, maxResults int) []ProductMatch {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if maxResults <= 0 {
		maxResults = 3
	}

	var results []ProductMatch

	lower := strings.ToLower(query)
	matchedBrand := ""
	for _, brand := range knownBrands {
		if strings.Contains(lower, brand) {
			matchedBrand = brand
			break
		}
	}

	if matchedBrand != "" {
		results = append(results, b.searchInProducts(matchedBrand, lower, "goangkasa", b.GoAngkasaProducts)...)
		results = append(results, b.searchInProducts(matchedBrand, lower, "goflexi", b.GoFlexiProducts)...)
	}

	if len(results) < maxResults {
		results = append(results, b.searchAllBrands(lower, "goangkasa", b.GoAngkasaProducts, results)...)
	}
	if len(results) < maxResults {
		results = append(results, b.searchAllBrands(lower, "goflexi", b.GoFlexiProducts, results)...)
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Source != results[j].Source {
			return results[i].Source < results[j].Source
		}
		return results[i].Model < results[j].Model
	})

	if len(results) > maxResults {
		results = results[:maxResults]
	}

	return results
}

func (b *Brain) searchInProducts(brand, query, source string, products map[string]map[string]ProductData) []ProductMatch {
	var results []ProductMatch

	for brandSlug, models := range products {
		if !strings.Contains(strings.ToLower(brandSlug), brand) {
			continue
		}

		for modelSlug, pd := range models {
			if modelContainsQuery(modelSlug, pd.Model, query) {
				results = append(results, ProductMatch{
					Brand:  pd.Brand,
					Model:  pd.Model,
					Source: source,
					Data:   pd,
				})
			}
		}
	}

	return results
}

func (b *Brain) searchAllBrands(query, source string, products map[string]map[string]ProductData, existing []ProductMatch) []ProductMatch {
	var results []ProductMatch

	for brandSlug, models := range products {
		for modelSlug, pd := range models {
			if alreadyMatched(existing, brandSlug, modelSlug) {
				continue
			}
			if modelContainsQuery(modelSlug, pd.Model, query) {
				results = append(results, ProductMatch{
					Brand:  pd.Brand,
					Model:  pd.Model,
					Source: source,
					Data:   pd,
				})
			}
		}
	}

	return results
}

func modelContainsQuery(modelSlug, modelDisplay, query string) bool {
	lowerSlug := strings.ToLower(modelSlug)
	lowerModel := strings.ToLower(modelDisplay)
	lowerQuery := strings.ToLower(query)

	words := strings.Fields(lowerQuery)
	for _, word := range words {
		if len(word) < 2 {
			continue
		}
		if strings.Contains(lowerSlug, word) || strings.Contains(lowerModel, word) {
			return true
		}
	}

	for _, brand := range knownBrands {
		if strings.Contains(lowerQuery, brand) {
			lowerQuery = strings.ReplaceAll(lowerQuery, brand, "")
		}
	}
	lowerQuery = strings.TrimSpace(lowerQuery)

	if lowerQuery != "" {
		queryWords := strings.Fields(lowerQuery)
		for _, word := range queryWords {
			if len(word) < 2 {
				continue
			}
			if strings.Contains(lowerSlug, word) || strings.Contains(lowerModel, word) {
				return true
			}
		}
	}

	return false
}

func alreadyMatched(existing []ProductMatch, brandSlug, modelSlug string) bool {
	for _, m := range existing {
		lowerBrand := strings.ToLower(m.Data.BrandSlug)
		if lowerBrand == "" {
			lowerBrand = strings.ToLower(m.Brand)
		}
		lowerModel := strings.ToLower(m.Data.ModelSlug)
		if lowerModel == "" {
			lowerModel = strings.ToLower(m.Model)
		}
		if lowerBrand == strings.ToLower(brandSlug) && lowerModel == strings.ToLower(modelSlug) {
			return true
		}
	}
	return false
}

// FormatProduct formats a ProductMatch into a human-readable text block
// suitable for inclusion in an AI prompt. It includes brand, model,
// storage variants, and representative installment prices (lowest, mid, shortest).
func (b *Brain) FormatProduct(p ProductMatch) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Brand: %s\n", p.Data.Brand))
	sb.WriteString(fmt.Sprintf("Model: %s\n", p.Data.Model))
	sb.WriteString(fmt.Sprintf("Plan: %s\n", p.Source))

	if len(p.Data.Variants) == 0 {
		return sb.String()
	}

	sb.WriteString("Variants:\n")

	for idx, variant := range p.Data.Variants {
		title := getStr(variant, "title")
		storage := getStr(variant, "storage")
		label := title
		if label == "" && storage != "" {
			label = storage
		}
		if label == "" {
			label = fmt.Sprintf("Variant %d", idx+1)
		}

		sb.WriteString(fmt.Sprintf("  - %s\n", label))

		prices := extractInstallmentPrices(variant)
		if len(prices) > 0 {
			summary := summarizePrices(prices)
			sb.WriteString(fmt.Sprintf("    Installment: %s\n", summary))
		}
	}

	return sb.String()
}

func extractInstallmentPrices(variant map[string]interface{}) map[string]float64 {
	prices := make(map[string]float64)

	if plans, ok := variant["installmentPlans"]; ok {
		if plansMap, ok := plans.(map[string]interface{}); ok {
			for tenure, val := range plansMap {
				if f, ok := toFloat(val); ok && f > 0 {
					prices[tenure] = f
				}
			}
		}
	}

	if amount, ok := variant["amount"]; ok {
		if amountMap, ok := amount.(map[string]interface{}); ok {
			if val, ok := toFloat(amountMap["value"]); ok && val > 0 {
				prices["monthly"] = val
			}
		}
	}

	return prices
}

func summarizePrices(prices map[string]float64) string {
	if len(prices) == 0 {
		return ""
	}

	vals := make([]float64, 0, len(prices))
	for _, v := range prices {
		vals = append(vals, v)
	}
	sort.Float64s(vals)

	if len(vals) == 1 {
		return fmt.Sprintf("RM%.2f/month", vals[0])
	}

	lowest := vals[0]
	highest := vals[len(vals)-1]
	mid := vals[len(vals)/2]

	parts := []string{
		fmt.Sprintf("from RM%.0f", lowest),
		fmt.Sprintf("mid ~RM%.0f", mid),
		fmt.Sprintf("up to RM%.0f", highest),
	}

	return strings.Join(parts, ", ")
}

func getStr(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func toFloat(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case int32:
		return float64(n), true
	default:
		return 0, false
	}
}
