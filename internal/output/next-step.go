package output

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/transcript1-com/cli/internal/api"
	"github.com/transcript1-com/cli/internal/inputs"
)

const (
	appURL       = "https://transcript1.com"
	billingURL   = appURL + "/dashboard/billing"
	tokensURL    = appURL + "/dashboard/account/tokens"
	supportEmail = "support@transcript1.com"

	signInStep   = "Run transcript1 login or set TRANSCRIPT1_API_KEY. Keys: " + tokensURL
	runAgainStep = "Run the command again"
	fallbackStep = "Retry later; if it keeps failing, contact " + supportEmail
	haveFileStep = "if you have the file, run transcript1 <file>"

	platformNames = "YouTube, X, TikTok, Instagram, Facebook or Bluesky"
)

var defaultPlatforms = []string{"youtube", "x", "tiktok", "instagram", "facebook", "bluesky"}

var steps = map[string]func(e *api.Error) string{
	api.CodeInvalidRequest: func(*api.Error) string {
		return "Check the input and options (transcript1 --help), then try again"
	},
	api.CodeInvalidURL: func(*api.Error) string {
		return "Use a " + platformNames + " link to a video or post, not a profile or playlist"
	},
	api.CodeUnsupportedPlatform: unsupportedPlatform,
	api.CodeUnsupportedMedia: func(*api.Error) string {
		return "Use a file with an audio track in one of these formats: " + strings.Join(inputs.Extensions, ", ")
	},
	api.CodeUnauthorized:      func(*api.Error) string { return signInStep },
	api.CodeInsufficientScope: func(*api.Error) string { return signInStep },
	api.CodeEmailNotVerified: func(*api.Error) string {
		return "Verify your email address with the link we sent, or request a new link at " + appURL + "/verify-resend"
	},
	api.CodeInsufficientMinutes: func(e *api.Error) string { return "Top up at " + actionURL(e) },
	api.CodeNoActivePlan:        func(e *api.Error) string { return "Choose a plan at " + actionURL(e) },
	api.CodeNotFound:            func(*api.Error) string { return "Check the id (it starts with tr_)" },
	api.CodeUploadIncomplete:    func(*api.Error) string { return runAgainStep },
	api.CodeInvalidState:        func(*api.Error) string { return runAgainStep },
	api.CodeIdempotencyConflict: func(*api.Error) string { return runAgainStep },
	api.CodeUploadTokenRefused:  func(*api.Error) string { return runAgainStep + "; the upload starts over" },
	api.CodeMediaTooLarge:       mediaTooLarge,
	api.CodeRateLimited:         rateLimited,
	api.CodeTooManyActiveJobs: func(*api.Error) string {
		return "Wait for a running transcription to finish, or lower --jobs"
	},
	api.CodeInternalError: func(*api.Error) string {
		return "Retry in a few seconds; if it keeps failing, contact " + supportEmail
	},
	api.CodeServiceUnavailable: func(*api.Error) string {
		return "transcript1 is temporarily unavailable; retry in a minute"
	},
	api.CodeMediaUnavailable: func(*api.Error) string {
		return "Open the " + platformNames + " link and check that the video plays without signing in"
	},
	api.CodeMediaPrivate: func(*api.Error) string {
		return "Only public or unlisted videos can be fetched; " + haveFileStep
	},
	api.CodeLoginRequired: func(*api.Error) string {
		return "Only public posts can be fetched; " + haveFileStep
	},
	api.CodeAgeRestricted: func(*api.Error) string {
		return "Age-restricted videos cannot be fetched; " + haveFileStep
	},
	api.CodeLiveStream: func(*api.Error) string {
		return "Live and upcoming streams cannot be transcribed; retry after the stream has ended"
	},
	api.CodeNoAudio: func(*api.Error) string {
		return "The video has no sound track (nothing was charged); pick a post with audio"
	},
	api.CodeNoCaptions: func(*api.Error) string { return "Run again with --source auto or --source stt" },
	api.CodeMediaTooLong: func(*api.Error) string {
		return "Speech-to-text accepts up to 5 hours; split the media and transcribe the parts"
	},
	api.CodeNoSpeech: func(*api.Error) string {
		return "No speech was detected (nothing was charged); check that the media has audible speech"
	},
	api.CodeUnsupportedLanguage: func(*api.Error) string {
		return "Set --language to a supported code or auto, or check the language spoken in the media"
	},
	api.CodeFetchFailed: func(*api.Error) string {
		return "Fetching the media failed after several attempts (nothing was charged); retry later"
	},
	api.CodeTranscriptionFailed: func(*api.Error) string {
		return "Speech-to-text failed after several attempts (nothing was charged); retry later"
	},
}

func NextStep(e *api.Error) string {
	if e == nil {
		return ""
	}

	if step, ok := steps[e.Code]; ok {
		return step(e)
	}

	return fallbackStep
}

func actionURL(e *api.Error) string {
	url, ok := e.Details["action_url"].(string)
	if !ok {
		return billingURL
	}

	if url == "" {
		return billingURL
	}

	return url
}

func listPlatforms(e *api.Error) []string {
	list, ok := e.Details["supported_platforms"].([]any)
	if !ok {
		return slices.Clone(defaultPlatforms)
	}

	if len(list) == 0 {
		return slices.Clone(defaultPlatforms)
	}

	platforms := []string{}
	for _, item := range list {
		platforms = append(platforms, fmt.Sprint(item))
	}

	return platforms
}

func unsupportedPlatform(e *api.Error) string {
	return "Supported platforms: " + strings.Join(listPlatforms(e), ", ") + ". For a local file, pass its path"
}

func mediaTooLarge(e *api.Error) string {
	limit, ok := e.Details["max_bytes"].(float64)
	if !ok {
		return "Use a smaller file"
	}

	bytes := strconv.FormatFloat(limit, 'f', -1, 64)

	return "The maximum file size is " + bytes + " bytes; compress or split the file"
}

func rateLimited(e *api.Error) string {
	if e.RetryAfter <= 0 {
		return "Wait a minute, then run the command again"
	}

	return fmt.Sprintf("Wait %d seconds, then run the command again", int(e.RetryAfter.Seconds()))
}
