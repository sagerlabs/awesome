package agent

// buildFeedbackInstruction appends a coaching directive based on the previous-turn feedback signal.
func buildFeedbackInstruction(feedback *AdviceFeedback) string {
	if feedback == nil {
		return ""
	}
	switch feedback.Type {
	case FeedbackRejected:
		return "用户已表示上一轮建议没命中。开头用一句话承认并重新判断，不要重复上一轮原话；如果知识库证据不足，要直接说明不足。"
	case FeedbackNeedsContext:
		return "用户补充了新局面。先说明这个上下文会改变判断，并基于新阶段/血量/装备重新给操作。"
	case FeedbackAcceptedOrContinued:
		return "用户沿着上一轮继续追问。不要重新科普整套阵容，直接推进下一步操作。"
	default:
		return ""
	}
}

func formatAdviceFeedback(feedbackType string) string {
	switch feedbackType {
	case FeedbackRejected:
		return "上一轮建议可能没命中"
	case FeedbackNeedsContext:
		return "用户补充了新局面"
	case FeedbackAcceptedOrContinued:
		return "用户沿着上一轮继续追问"
	default:
		return feedbackType
	}
}
