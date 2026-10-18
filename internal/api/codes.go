package api

const (
	CodeInvalidRequest      = "invalid_request"
	CodeInvalidURL          = "invalid_url"
	CodeUnsupportedPlatform = "unsupported_platform"
	CodeUnsupportedMedia    = "unsupported_media"
	CodeUnauthorized        = "unauthorized"
	CodeInsufficientScope   = "insufficient_scope"
	CodeEmailNotVerified    = "email_not_verified"
	CodeInsufficientMinutes = "insufficient_minutes"
	CodeNoActivePlan        = "no_active_plan"
	CodeNotFound            = "not_found"
	CodeUploadIncomplete    = "upload_incomplete"
	CodeInvalidState        = "invalid_state"
	CodeIdempotencyConflict = "idempotency_conflict"
	CodeMediaTooLarge       = "media_too_large"
	CodeRateLimited         = "rate_limited"
	CodeTooManyActiveJobs   = "too_many_active_jobs"
	CodeInternalError       = "internal_error"
	CodeServiceUnavailable  = "service_unavailable"
)

const (
	CodeMediaUnavailable    = "media_unavailable"
	CodeMediaPrivate        = "media_private"
	CodeAgeRestricted       = "age_restricted"
	CodeLiveStream          = "live_stream"
	CodeNoCaptions          = "no_captions"
	CodeMediaTooLong        = "media_too_long"
	CodeNoSpeech            = "no_speech"
	CodeUnsupportedLanguage = "unsupported_language"
	CodeFetchFailed         = "fetch_failed"
	CodeTranscriptionFailed = "transcription_failed"
)

const (
	CodeUploadTokenRefused = "upload_token_refused"
)

const (
	CodeValidationError    = "validation_error"
	CodeRateLimitExceeded  = "rate_limit_exceeded"
	CodeCliLoginNotFound   = "cli_login_not_found"
	CodeCliLoginDenied     = "cli_login_denied"
	CodeCliLoginExpired    = "cli_login_expired"
	CodeCliLoginConsumed   = "cli_login_consumed"
	CodePatLimitReached    = "pat_limit_reached"
	CodePatNotFound        = "pat_not_found"
	CodeAuthInvalidSession = "auth_invalid_session"
)

const (
	LoginPending  = "pending"
	LoginSlowDown = "slow_down"
	LoginApproved = "approved"
)
