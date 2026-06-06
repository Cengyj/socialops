package service

import (
	"context"
	"strings"

	dbent "github.com/Wei-Shaw/socialops/ent"
	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
	"github.com/Wei-Shaw/socialops/internal/pkg/socialidentity"
)

type SocialAccountCredentialRequest struct {
	Name          string
	Password      string
	Email         string
	EmailPassword string
	ProxyEndpoint string
}

type SocialAccountCredentialResult struct {
	ExecutionAuth  string
	PlatformUserID *string
	Message        string
}

type SocialAccountCredentialRegistrar interface {
	Supports(platform string) bool
	AcquireCredentials(ctx context.Context, req SocialAccountCredentialRequest) (*SocialAccountCredentialResult, error)
}

type RegisterSocialAccountInput struct {
	Name           string  `json:"name" binding:"required"`
	Platform       string  `json:"platform" binding:"required"`
	PlatformUserID *string `json:"platform_user_id"`
	Password       *string `json:"password"`
	Phone          *string `json:"phone"`
	Email          *string `json:"email"`
	EmailPassword  *string `json:"email_password"`
	Remark         *string `json:"remark"`
}

func (s *SocialAccountService) RegisterWithCredentials(ctx context.Context, input *RegisterSocialAccountInput, registrar SocialAccountCredentialRegistrar, proxyEndpoint string) (*SocialAccount, error) {
	if input == nil {
		return nil, infraerrors.BadRequest("SOCIAL_ACCOUNT_INPUT_REQUIRED", "social account input is required")
	}
	if registrar == nil {
		return nil, infraerrors.ServiceUnavailable("SOCIAL_ACCOUNT_REGISTRAR_NOT_CONFIGURED", "social account registrar is not configured; no account was created")
	}
	name := strings.TrimSpace(input.Name)
	platform := normalizeSocialPlatform(input.Platform)
	if name == "" {
		return nil, infraerrors.BadRequest("SOCIAL_ACCOUNT_NAME_REQUIRED", "social account name is required")
	}
	if platform == "" {
		return nil, infraerrors.BadRequest("SOCIAL_ACCOUNT_PLATFORM_REQUIRED", "social account platform is required")
	}
	if !registrar.Supports(platform) {
		return nil, infraerrors.ServiceUnavailable("SOCIAL_ACCOUNT_REGISTRAR_NOT_CONFIGURED", "social account registrar is not configured for this platform")
	}
	password := optionalStringValue(input.Password)
	if password == "" {
		return nil, infraerrors.BadRequest("SOCIAL_ACCOUNT_PASSWORD_REQUIRED", "social account password is required")
	}
	identity := socialAccountBusinessIdentity(platform, name)
	exists, err := s.poolAccountExists(ctx, identity)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrSocialAccountDuplicate
	}

	account, err := s.createRegisteringAccount(ctx, input, name, platform, identity)
	if err != nil {
		return nil, err
	}

	result, err := registrar.AcquireCredentials(ctx, SocialAccountCredentialRequest{
		Name:          name,
		Password:      password,
		Email:         optionalStringValue(input.Email),
		EmailPassword: optionalStringValue(input.EmailPassword),
		ProxyEndpoint: strings.TrimSpace(proxyEndpoint),
	})
	if err != nil {
		return s.markRegisteredAccountFailed(ctx, account.ID, err)
	}
	executionAuth := ""
	var platformUserID *string
	if result != nil {
		executionAuth = strings.TrimSpace(result.ExecutionAuth)
		platformUserIDValue := trimPtr(result.PlatformUserID)
		if platformUserIDValue != "" {
			platformUserID = &platformUserIDValue
		}
	}
	if executionAuth == "" {
		return s.markRegisteredAccountFailed(ctx, account.ID, newSocialExecutionError(SocialExecutionFailureAuthInvalid, "login succeeded without usable auth credentials", nil))
	}
	message := strings.TrimSpace(result.Message)
	if message == "" {
		message = "login succeeded"
	}
	ent, err := s.entClient.SocialAccount.UpdateOneID(account.ID).
		SetExecutionAuth(executionAuth).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SetTaskMessage(message).
		SetNillablePlatformUserID(platformUserID).
		Save(ctx)
	if err != nil {
		if dbent.IsConstraintError(err) {
			return nil, ErrSocialAccountDuplicate
		}
		return nil, err
	}
	return socialAccountFromEnt(ent), nil
}

func (s *SocialAccountService) createRegisteringAccount(ctx context.Context, input *RegisterSocialAccountInput, name, platform string, identity socialidentity.BusinessIdentity) (*dbent.SocialAccount, error) {
	q := s.entClient.SocialAccount.Create().
		SetName(name).
		SetPlatform(platform).
		SetPlatformKey(identity.PlatformKey).
		SetNameKey(identity.NameKey).
		SetIdentityKind(identity.IdentityKind).
		SetIdentityKey(identity.IdentityKey).
		SetAccountStatus(SocialAccountStatusPendingCheck).
		SetTaskStatus(SocialTaskStatusRegistering)

	if value := trimPtr(input.PlatformUserID); value != "" {
		q.SetPlatformUserID(value)
	}
	if input.Password != nil {
		q.SetPassword(*input.Password)
	}
	if input.Phone != nil {
		q.SetPhone(strings.TrimSpace(*input.Phone))
	}
	if input.Email != nil {
		q.SetEmail(strings.TrimSpace(*input.Email))
	}
	if input.EmailPassword != nil {
		q.SetEmailPassword(*input.EmailPassword)
	}
	if input.Remark != nil {
		q.SetRemark(*input.Remark)
	}
	ent, err := q.Save(ctx)
	if err != nil {
		if dbent.IsConstraintError(err) {
			return nil, ErrSocialAccountDuplicate
		}
		return nil, err
	}
	return ent, nil
}

func (s *SocialAccountService) markRegisteredAccountFailed(ctx context.Context, accountID int64, failure error) (*SocialAccount, error) {
	accountStatus, taskStatus := socialAccountRegistrationFailureState(failure)
	message := "login failed"
	if failure != nil && strings.TrimSpace(failure.Error()) != "" {
		message = failure.Error()
	}
	ent, err := s.entClient.SocialAccount.UpdateOneID(accountID).
		SetAccountStatus(accountStatus).
		SetTaskStatus(taskStatus).
		SetTaskMessage(message).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return socialAccountFromEnt(ent), nil
}

func socialAccountRegistrationFailureState(err error) (accountStatus, taskStatus string) {
	kind, ok := socialExecutionFailureKind(err)
	if !ok {
		return SocialAccountStatusPendingCheck, SocialTaskStatusRegisterFailed
	}
	switch kind {
	case SocialExecutionFailureAuthMissing:
		return SocialAccountStatusNotStored, SocialTaskStatusManualReview
	case SocialExecutionFailureAuthInvalid:
		return SocialAccountStatusInvalid, SocialTaskStatusRegisterFailed
	case SocialExecutionFailureAccountLimited:
		return SocialAccountStatusLimited, SocialTaskStatusManualReview
	case SocialExecutionFailureChallengeRequired:
		return SocialAccountStatusPendingCheck, SocialTaskStatusManualReview
	case SocialExecutionFailureProxyMissing, SocialExecutionFailureProxyInvalid, SocialExecutionFailureProxyUnavailable, SocialExecutionFailureNetwork:
		return SocialAccountStatusPendingCheck, SocialTaskStatusIPUnavailable
	default:
		return SocialAccountStatusPendingCheck, SocialTaskStatusRegisterFailed
	}
}

func optionalStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
