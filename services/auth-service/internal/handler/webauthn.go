package handler

// WebAuthn verification is now handled by the service layer.
// The handler layer calls AuthService.VerifyRegistration and
// AuthService.VerifyAssertion directly, which internally use
// the go-webauthn library for real cryptographic verification.
//
// Dev mode bypass (device_public_key starting with "dev:") is
// preserved in the service layer for testing purposes.