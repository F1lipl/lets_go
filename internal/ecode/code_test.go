package ecode

import "testing"

func TestCodesAreUniqueAndHaveMessages(t *testing.T) {
	codes := []Code{
		Success,
		InvalidRequest,
		InvalidUsername,
		InvalidPhoneNumber,
		InvalidPassword,
		UserNotFound,
		PhoneAlreadyRegistered,
		UsernameAlreadyExists,
		AccountDisabled,
		AccountPending,
		LoginCredentialInvalid,
		AccessTokenInvalid,
		AccessTokenExpired,
		RefreshTokenInvalid,
		RefreshTokenExpired,
		VerificationCodeInvalid,
		VerificationCodeExpired,
		VerificationCodeTooFrequent,
		InvalidDeviceInfo,
		DeviceNotFound,
		DeviceDisabled,
		SessionNotFound,
		SessionInactive,
		SessionExpired,
		SessionLimitExceeded,
		InternalError,
		DatabaseError,
		CacheError,
	}

	seen := make(map[Code]struct{}, len(codes))
	for _, code := range codes {
		if _, exists := seen[code]; exists {
			t.Fatalf("duplicate code: %d", code)
		}
		seen[code] = struct{}{}

		if _, exists := messages[code]; !exists {
			t.Fatalf("missing message for code: %d", code)
		}
	}
}

func TestUnknownCodeUsesInternalErrorMessage(t *testing.T) {
	unknown := Code(123456789)
	if got, want := unknown.Message(), InternalError.Message(); got != want {
		t.Fatalf("unexpected fallback message: got %q, want %q", got, want)
	}
}
