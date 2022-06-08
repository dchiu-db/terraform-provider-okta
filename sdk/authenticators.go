package sdk

// List of factors that are applicable to Okta Identity Engine (OIE)
var AuthenticatorProviders = []string{
	// NOTE: some authenticator types are available by feature flag on the org only
	DuoFactor,
	ExternalIdpFactor,
	GoogleOtpFactor,
	OktaEmailFactor,
	OktaPasswordFactor, // NOTE: Not configurable in OIE policies (Handle downstream as necessary)
	OktaVerifyFactor,
	OnPremMfaFactor,
	PhoneNumberFactor,
	RsaTokenFactor,
	SecurityQuestionFactor,
	WebauthnFactor,
    // YubikeyTokenFactor, // NOTE: support upcoming when available in public API
}
