package brain

import (
	"strings"
)

var intentKeywords = map[string][]string{
	"greeting": {
		"hi", "hello", "hey", "hai", "haiya", "helo", "hiya",
		"good morning", "good afternoon", "good evening",
		"selamat pagi", "selamat petang", "selamat malam",
		"apa khabar", "greetings",
	},
	"product_inquiry": {
		"iphone", "samsung", "galaxy", "xiaomi", "vivo", "oppo", "realme",
		"honor", "asus", "iqoo", "tecno", "oneplus", "nothing", "red magic",
		"infinix", "ipad", "tablet", "phone", "phone ada", "handphone",
		"available", "ada tak", "ada x", "stock", "ada ke",
		"mac", "laptop", "pixel", "google pixel",
		"pro max", "pro", "plus", "ultra", "mini", "se",
		"note", "redmi", "poco", "rog",
	},
	"plan_comparison": {
		"goangkasa", "goflexi", "angkasa", "flexi",
		"plan", "plans", "salary deduction", "potongan gaji",
		"installment plan", "installment", "perbezaan", "bezanya",
		"which plan", "compare", "banding", "comparison",
		"yuran", "interest", "deposit", "tipuan",
	},
	"application": {
		"apply", "daftar", "register", "sign up", "apply now",
		"mohon", "nak apply", "want to apply", "how to apply",
		"macam mana nak", "how to register", "nak mohon",
		"boleh apply", "can apply", "apply macam mana",
		"order", "nak order", "want to order",
	},
	"eligibility": {
		"eligible", "kelayakan", "layak", "boleh tak",
		"can i apply", "approval", "approve", "lulus",
		"ctos", "ccris", "blacklist", "bankrupt",
		"government staff", "glc", "koperasi", "cooperative",
		"kerajaan", "swasta", "sector", "potongan angkasa",
	},
	"pricing": {
		"price", "harga", "berapa", "how much", "cost",
		"monthly", "bulanan", "installment", "bayaran",
		"monthly berapa", "berapa sebulan", "budget",
		"murah", "mahal", "expensive", "cheap",
		"bawah", "below", "rm", "ringgit",
	},
	"objection": {
		"too expensive", "mahal", "tak mampu", "cannot afford",
		"too much", "too high", "not worth it", "tak berbaloi",
		"think about it", "consider", "nanti dulu",
		"not interested", "tak berminat", "no need",
	},
	"follow_up": {
		"thanks", "thank you", "terima kasih", "tq", "ty",
		"ok", "okay", "baik", "got it", "noted",
		"will do", "sure", "of course",
	},
	"escalation": {
		"human", "agent", "real person", "orang",
		"manager", "supervisor", "complaint", "aduan",
		"speak to someone", "talk to someone", "customer service",
	},
}

var intentFlowMap = map[string]string{
	"greeting":        "greeting",
	"product_inquiry": "product_inquiry",
	"plan_comparison": "plan_comparison",
	"application":     "application_flow",
	"eligibility":     "application_flow",
	"pricing":         "product_inquiry",
	"objection":       "objection_handling",
	"follow_up":       "follow_up",
	"escalation":      "escalation",
	"general":         "greeting",
}

// ClassifyIntent analyzes the user message and returns the detected intent string.
// It uses keyword matching with weighted scoring — longer multi-word keywords
// score higher to reduce false positives from short common words.
func (b *Brain) ClassifyIntent(message string) string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return classifyIntent(message)
}

func classifyIntent(message string) string {
	lower := strings.ToLower(message)

	bestIntent := "general"
	bestScore := 0

	for intent, keywords := range intentKeywords {
		score := 0
		for _, kw := range keywords {
			if strings.Contains(lower, kw) {
				score += len(strings.Fields(kw))
			}
		}
		if score > bestScore {
			bestScore = score
			bestIntent = intent
		}
	}

	return bestIntent
}

// ResolveFlow maps a classified intent to its corresponding flow filename.
func (b *Brain) ResolveFlow(intent string) string {
	if flow, ok := intentFlowMap[intent]; ok {
		return flow
	}
	return "greeting"
}
