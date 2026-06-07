package service

import "errors"

// SocialExecutionFailureKind describes the operational reason a real social
// action failed. It lets the SocialOps task worker update account health
// without coupling itself to a platform adapter's text messages.
type SocialExecutionFailureKind string

const (
	SocialExecutionFailureAuthMissing       SocialExecutionFailureKind = "auth_missing"
	SocialExecutionFailureAuthInvalid       SocialExecutionFailureKind = "auth_invalid"
	SocialExecutionFailureProxyMissing      SocialExecutionFailureKind = "proxy_missing"
	SocialExecutionFailureProxyInvalid      SocialExecutionFailureKind = "proxy_invalid"
	SocialExecutionFailureProxyUnavailable  SocialExecutionFailureKind = "proxy_unavailable"
	SocialExecutionFailureNetwork           SocialExecutionFailureKind = "network"
	SocialExecutionFailureAccountLimited    SocialExecutionFailureKind = "account_limited"
	SocialExecutionFailureChallengeRequired SocialExecutionFailureKind = "challenge_required"
	SocialExecutionFailureActionInput       SocialExecutionFailureKind = "action_input"
	SocialExecutionFailurePlatform          SocialExecutionFailureKind = "platform"
	SocialExecutionFailureUnsupported       SocialExecutionFailureKind = "unsupported"
)

// SocialExecutionError wraps a business-safe failure message with a structured
// failure kind for internal task/account lifecycle handling.
type SocialExecutionError struct {
	Kind    SocialExecutionFailureKind
	Message string
	Cause   error
}

func (e *SocialExecutionError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Cause != nil {
		return e.Cause.Error()
	}
	return string(e.Kind)
}

func (e *SocialExecutionError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func newSocialExecutionError(kind SocialExecutionFailureKind, message string, cause error) error {
	if message == "" && cause != nil {
		message = cause.Error()
	}
	return &SocialExecutionError{Kind: kind, Message: message, Cause: cause}
}

func socialExecutionFailureKind(err error) (SocialExecutionFailureKind, bool) {
	var execErr *SocialExecutionError
	if errors.As(err, &execErr) && execErr != nil && execErr.Kind != "" {
		return execErr.Kind, true
	}
	return "", false
}
