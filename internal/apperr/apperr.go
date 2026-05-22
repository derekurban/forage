package apperr

import "fmt"

const (
	CodeGeneral             = "general_failure"
	CodeInvalidConfig       = "invalid_config"
	CodeConfigMissing       = "config_missing"
	CodeAuthMissing         = "auth_missing"
	CodeProviderUnavailable = "provider_unavailable"
	CodeCapabilityExhausted = "capability_exhausted"
	CodeCacheMiss           = "cache_miss"
	CodeNotImplemented      = "not_implemented"
)

const (
	ExitOK                  = 0
	ExitGeneral             = 1
	ExitInvalidArgsOrConfig = 2
	ExitCapabilityExhausted = 3
	ExitNoProvider          = 4
	ExitAuthSetup           = 5
	ExitPolicyBlock         = 6
	ExitCacheMiss           = 7
	ExitPartialSuccess      = 8
)

type Error struct {
	Code       string         `json:"code"`
	Message    string         `json:"message"`
	Capability string         `json:"capability,omitempty"`
	RetryAfter string         `json:"retry_after,omitempty"`
	Providers  []ProviderInfo `json:"providers,omitempty"`
	ExitCode   int            `json:"-"`
}

type ProviderInfo struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Reason     string `json:"reason,omitempty"`
	RetryAfter string `json:"retry_after,omitempty"`
}

func (e *Error) Error() string {
	return e.Message
}

func New(code, message string, exitCode int) *Error {
	return &Error{Code: code, Message: message, ExitCode: exitCode}
}

func MissingConfig(path string) *Error {
	return New(CodeConfigMissing, fmt.Sprintf("Missing config at %s. Run `forage config init` or `forage setup`.", path), ExitInvalidArgsOrConfig)
}
