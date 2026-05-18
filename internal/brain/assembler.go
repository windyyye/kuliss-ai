package brain

import (
	"fmt"
	"strings"
)

const maxPromptChars = 16000

// AssemblePrompt builds a context-aware system prompt for the given user message.
// It selects relevant flows, knowledge, and product data based on intent classification.
// If the Brain was not loaded from AI_BRAIN, it returns the fallback prompt.
func (b *Brain) AssemblePrompt(userMessage string) string {
	if !b.Loaded {
		return b.FallbackPrompt
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	var sb strings.Builder

	sb.WriteString(b.MasterPrompt)

	if identity, ok := b.Core["identity"]; ok {
		sb.WriteString("\n\n## Core Identity\n")
		sb.WriteString(identity)
	}

	intent := classifyIntent(userMessage)
	flowName := b.resolveFlowLocked(intent)

	if flow, ok := b.Flows[flowName]; ok {
		sb.WriteString("\n\n## Current Task\n")
		sb.WriteString(flow)
	}

	switch intent {
	case "product_inquiry", "pricing":
		b.appendProductContext(&sb, userMessage)
	case "plan_comparison":
		b.appendPlanComparison(&sb)
	case "application", "eligibility":
		b.appendApplicationContext(&sb)
	case "objection":
		if dad, ok := b.Core["do_and_dont"]; ok {
			sb.WriteString("\n\n## Guidelines\n")
			sb.WriteString(dad)
		}
	}

	if style, ok := b.Core["communication_style"]; ok {
		sb.WriteString("\n\n## Communication Style\n")
		sb.WriteString(style)
	}

	result := sb.String()
	if len(result) > maxPromptChars {
		result = truncatePrompt(result, maxPromptChars)
	}

	return result
}

// GetFullPrompt returns the full assembled prompt without any user message context.
// This is intended for display purposes (e.g., the frontend Prompt page).
func (b *Brain) GetFullPrompt() string {
	if !b.Loaded {
		return b.FallbackPrompt
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	var sb strings.Builder

	sb.WriteString(b.MasterPrompt)

	if identity, ok := b.Core["identity"]; ok {
		sb.WriteString("\n\n## Core Identity\n")
		sb.WriteString(identity)
	}

	for name, flow := range b.Flows {
		sb.WriteString(fmt.Sprintf("\n\n## Flow: %s\n", name))
		sb.WriteString(flow)
	}

	if style, ok := b.Core["communication_style"]; ok {
		sb.WriteString("\n\n## Communication Style\n")
		sb.WriteString(style)
	}

	return sb.String()
}

func (b *Brain) appendProductContext(sb *strings.Builder, userMessage string) {
	matches := searchProductsLocked(b, userMessage, 3)

	for _, match := range matches {
		formatted := b.FormatProduct(match)
		sb.WriteString(fmt.Sprintf("\n\n## Product: %s %s (%s)\n", match.Brand, match.Model, match.Source))
		sb.WriteString(formatted)
	}

	if len(matches) > 0 {
		sources := make(map[string]bool)
		for _, m := range matches {
			sources[m.Source] = true
		}

		if sources["goangkasa"] {
			b.writeOverview(sb, b.GoAngkasaInfo, "GoAngkasa")
		}
		if sources["goflexi"] {
			b.writeOverview(sb, b.GoFlexiInfo, "GoFlexi")
		}
	}
}

func (b *Brain) appendPlanComparison(sb *strings.Builder) {
	b.writeOverview(sb, b.GoAngkasaInfo, "GoAngkasa")
	b.writeOverview(sb, b.GoFlexiInfo, "GoFlexi")
	if info, ok := b.Knowledge["company_info"]; ok {
		sb.WriteString("\n\n## Company Information\n")
		sb.WriteString(info)
	}
}

func (b *Brain) appendApplicationContext(sb *strings.Builder) {
	if elig, ok := b.Knowledge["eligibility"]; ok {
		sb.WriteString("\n\n## Eligibility Requirements\n")
		sb.WriteString(elig)
	}
	if process, ok := b.GoAngkasaInfo["goangkasa_process"]; ok {
		sb.WriteString("\n\n## GoAngkasa Application Process\n")
		sb.WriteString(process)
	}
	if process, ok := b.GoFlexiInfo["goflexi_process"]; ok {
		sb.WriteString("\n\n## GoFlexi Application Process\n")
		sb.WriteString(process)
	}
}

func (b *Brain) writeOverview(sb *strings.Builder, info map[string]string, planName string) {
	for key, content := range info {
		sb.WriteString(fmt.Sprintf("\n\n## %s: %s\n", planName, key))
		sb.WriteString(content)
	}
}

func (b *Brain) resolveFlowLocked(intent string) string {
	if flow, ok := intentFlowMap[intent]; ok {
		return flow
	}
	return "greeting"
}

func searchProductsLocked(b *Brain, query string, maxResults int) []ProductMatch {
	lower := strings.ToLower(query)
	matchedBrand := ""
	for _, brand := range knownBrands {
		if strings.Contains(lower, brand) {
			matchedBrand = brand
			break
		}
	}

	var results []ProductMatch

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

	if len(results) > maxResults {
		results = results[:maxResults]
	}

	return results
}

func truncatePrompt(prompt string, maxLen int) string {
	if len(prompt) <= maxLen {
		return prompt
	}

	truncated := prompt[:maxLen]

	lastSection := strings.LastIndex(truncated, "\n\n## ")
	if lastSection > maxLen/2 {
		truncated = truncated[:lastSection]
	}

	truncated = strings.TrimSpace(truncated)
	truncated += "\n\n[Additional context truncated for length]"

	return truncated
}
