package service

import "strings"

type twitterTaskFailureTranslation struct {
	userMessage string
	kind        SocialExecutionFailureKind
}

func KnownTwitterTaskFailureMessage(message string) (string, bool) {
	failure, ok := knownTwitterTaskFailure(message)
	if !ok {
		return "", false
	}
	return failure.userMessage, true
}

func IsTwitterPlatformFailureMessage(message string) bool {
	normalized := normalizeTwitterFailureMessage(message)
	return strings.HasPrefix(normalized, "twitter error ") ||
		strings.HasPrefix(normalized, "twitter login error:")
}

func knownTwitterTaskFailure(message string) (twitterTaskFailureTranslation, bool) {
	detail, ok := twitterPlatformFailureDetail(message)
	if !ok {
		return twitterTaskFailureTranslation{}, false
	}
	return knownTwitterFailureDetail(detail)
}

func twitterTaskFailureKind(message string) (SocialExecutionFailureKind, bool) {
	failure, ok := knownTwitterTaskFailure(message)
	if !ok || failure.kind == "" {
		return "", false
	}
	return failure.kind, true
}

func knownTwitterFailureDetail(detail string) (twitterTaskFailureTranslation, bool) {
	switch normalizeTwitterFailureDetail(detail) {
	case "wrong password",
		"incorrect password",
		"invalid password",
		"password is incorrect",
		"the password you entered is incorrect",
		"password you entered is incorrect",
		"密码错误":
		return twitterTaskFailureTranslation{
			userMessage: "密码错误，本次未扣费",
			kind:        SocialExecutionFailurePasswordInvalid,
		}, true
	case "sorry, we could not find your account",
		"account not found",
		"user not found":
		return twitterTaskFailureTranslation{
			userMessage: "账号不存在，本次未扣费",
			kind:        SocialExecutionFailureAuthInvalid,
		}, true
	case "user must verify login",
		"verify login",
		"login verification required",
		"additional verification required",
		"challenge required",
		"captcha challenge required",
		"confirm your identity to continue":
		return twitterTaskFailureTranslation{
			userMessage: "账号需要额外验证，本次未扣费",
			kind:        SocialExecutionFailureChallengeRequired,
		}, true
	case "this request looks like it might be automated",
		"account suspended",
		"account locked",
		"rate limit exceeded",
		"too many attempts",
		"follow limit reached":
		return twitterTaskFailureTranslation{
			userMessage: "账号状态或频率受限，本次未扣费",
			kind:        SocialExecutionFailureAccountLimited,
		}, true
	case "status is a duplicate",
		"duplicate tweet",
		"content is too long",
		"tweet needs to be a bit shorter",
		"you have already favorited this status",
		"you have already retweeted this tweet":
		return twitterTaskFailureTranslation{
			userMessage: "内容或目标状态不符合平台要求，本次未扣费",
			kind:        SocialExecutionFailureActionInput,
		}, true
	case "invalid or expired token",
		"could not authenticate you",
		"authentication failed",
		"invalid credentials":
		return twitterTaskFailureTranslation{
			userMessage: "账号认证信息不可用，本次未扣费",
			kind:        SocialExecutionFailureAuthInvalid,
		}, true
	default:
		return twitterTaskFailureTranslation{}, false
	}
}

func twitterPlatformFailureDetail(message string) (string, bool) {
	normalized := normalizeTwitterFailureMessage(message)
	if detail, ok := strings.CutPrefix(normalized, "twitter login error:"); ok {
		return strings.TrimSpace(detail), true
	}
	if strings.HasPrefix(normalized, "twitter error ") {
		_, detail, ok := strings.Cut(normalized, ":")
		if !ok {
			return "", true
		}
		return strings.TrimSpace(detail), true
	}
	return "", false
}

func normalizeTwitterFailureDetail(detail string) string {
	normalized := normalizeTwitterFailureMessage(detail)
	return strings.TrimRight(normalized, ".!")
}

func normalizeTwitterFailureMessage(message string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(message)), " "))
}
