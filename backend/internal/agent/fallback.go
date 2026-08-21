package agent

import (
	"errors"
	"strings"
)

const MaxStepFallbackAnswer = "我已经完成了可确认的部分，但剩余步骤需要更多信息或稍后重试。"
const SafeFallbackAnswer = "这一步没有稳定完成，我先给出保守回复：当前请求暂时无法确认执行成功，你可以稍后重试或缩小问题范围。"

func NormalizeMaxLoopSteps(value int) int {
	if value <= 0 {
		return 6
	}
	if value > 12 {
		return 12
	}
	return value
}

func NormalizeRetryLimit(value int) int {
	if value < 0 {
		return 0
	}
	if value > 3 {
		return 3
	}
	return value
}

func IsRetryableLLMError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "missing api key") || strings.Contains(message, "missing model") {
		return false
	}
	return errors.Is(err, contextLengthErr{}) ||
		strings.Contains(message, "timeout") ||
		strings.Contains(message, "temporarily") ||
		strings.Contains(message, "rate limit") ||
		strings.Contains(message, "status 429") ||
		strings.Contains(message, "status 500") ||
		strings.Contains(message, "status 502") ||
		strings.Contains(message, "status 503") ||
		strings.Contains(message, "status 504")
}

type contextLengthErr struct{}

func (contextLengthErr) Error() string { return "context length exceeded" }
