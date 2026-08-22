package store

func normalizeToolApprovalPolicy(value string) string {
	switch value {
	case "never", "always":
		return value
	default:
		return "sensitive_only"
	}
}

func normalizeThinkingEffort(value string) string {
	switch value {
	case "low", "high":
		return value
	default:
		return "medium"
	}
}
