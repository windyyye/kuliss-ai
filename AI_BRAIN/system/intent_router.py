"""
Intent Router for GoAnsuran WhatsApp Chatbot.

Maps user messages (English and Malay) to intent categories
for routing to the appropriate knowledge area or flow.

Usage:
    from intent_router import route_intent
    intent = route_intent("I want an iPhone", {})
    # Returns: "product_inquiry"
"""

# Intent keyword mapping (English and Malay)
INTENT_KEYWORDS = {
    "greeting": [
        "hi", "hello", "hey", "helo", "assalamualaikum", "salam",
        "good morning", "good afternoon", "good evening", "selamat pagi",
        "selamat petang", "selamat malam", "apa khabar", "how are you",
    ],
    "product_inquiry": [
        "iphone", "samsung", "galaxy", "pixel", "oppo", "honor", "xiaomi",
        "phone", "handphone", "hp", "smartphone", "mobile",
        "iphone 15", "iphone 16", "s24", "s25", "pixel 9",
        "model", "brand", "android", "ios",
    ],
    "plan_comparison": [
        "goangkasa", "angkasa", "goflexi", "flexi", "compare", "comparison",
        "versus", "vs", "difference", "beza", "perbandingan", "which plan",
        "which is better", "recommend plan",
    ],
    "application": [
        "apply", "daftar", "sign up", "register", "want to apply",
        "how to apply", "how to sign up", "mula", "start",
        "subscribe", "langgan",
    ],
    "eligibility": [
        "eligible", "eligibility", "kelayakan", "boleh apply", "can i apply",
        "who can", "siapa boleh", "civil servant", "penjawat awam",
        "government", "kerajaan", "glc", "coop", "asbn",
    ],
    "pricing": [
        "price", "harga", "how much", "berapa", "cost", "kos",
        "monthly", "bulan", "installment", "ansuran",
        "deposit", "payment", "bayaran",
        "cheap", "murah", "affordable", "budget", "bajet",
    ],
    "objection": [
        "too expensive", "mahal", "tak mampu", "cannot afford",
        "not sure", "tak pasti", "prefer buying", "think about it",
        "not interested", "tak berminat", "too costly",
    ],
    "follow_up": [
        "thanks", "thank you", "terima kasih", "bye", "goodbye",
        "that's all", "that's it", "saja", "just browsing",
    ],
    "escalation": [
        "human", "agent", "orang", "real person", "complaint",
        "aduan", "refund", "pulangkan", "cancel", "batal",
        "legal", "saman", "lawyer", "peguam",
    ],
}


def route_intent(message, context):
    """
    Route a user message to an intent category.

    Args:
        message: User's message string
        context: Dict with user context (plan_interest, brand_preference, etc.)

    Returns:
        str: Intent category key
    """
    msg_lower = message.lower().strip()

    # Score each intent by counting keyword matches
    scores = {}
    for intent, keywords in INTENT_KEYWORDS.items():
        score = 0
        for keyword in keywords:
            if keyword in msg_lower:
                # Longer keyword matches get higher score
                score += len(keyword.split())
        if score > 0:
            scores[intent] = score

    # If no matches, check context for hints
    if not scores:
        if context.get("plan_interest"):
            return "product_inquiry"
        return "general"

    # Return highest scoring intent
    best_intent = max(scores, key=scores.get)
    return best_intent


def get_suggested_flow(intent):
    """Map an intent to a suggested flow file name."""
    flow_map = {
        "greeting": "greeting",
        "product_inquiry": "product_inquiry",
        "plan_comparison": "plan_comparison",
        "application": "application_flow",
        "eligibility": "application_flow",
        "pricing": "product_inquiry",
        "objection": "objection_handling",
        "follow_up": "follow_up",
        "escalation": "escalation",
        "general": "greeting",
    }
    return flow_map.get(intent, "greeting")


if __name__ == "__main__":
    import sys

    test_messages = [
        "hi there",
        "I want an iPhone",
        "how much is samsung s24",
        "goangkasa vs goflexi",
        "I want to apply",
        "am I eligible",
        "too expensive",
        "terima kasih",
        "saya nak bercakap dengan orang",
    ]
    for msg in test_messages:
        intent = route_intent(msg, {})
        flow = get_suggested_flow(intent)
        print(f'"{msg}" -> {intent} -> {flow}.txt')
