package distribution

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ReleaseRetry is advisory response metadata, never permission to retry an
// installation. It contains no provider body, URL or unvalidated header text.
type ReleaseRetry struct {
	HTTPStatus        int    `json:"http_status"`
	Kind              string `json:"kind"`
	RetryAfterSeconds int64  `json:"retry_after_seconds,omitempty"`
	ResetAt           string `json:"reset_at,omitempty"`
}

type ReleaseRateLimitError struct{ Retry ReleaseRetry }

func (e *ReleaseRateLimitError) Error() string {
	message := "The release request was rate-limited."
	if e.Retry.RetryAfterSeconds > 0 {
		message += fmt.Sprintf(" Wait at least %d seconds from this response.", e.Retry.RetryAfterSeconds)
	}
	if e.Retry.ResetAt != "" {
		message += " Reported quota reset: " + e.Retry.ResetAt + "."
	}
	if e.Retry.RetryAfterSeconds == 0 && e.Retry.ResetAt == "" {
		message += " Retry timing is unavailable; wait before explicitly retrying."
	} else {
		message += " Do not retry before the reported limits; availability afterward is not guaranteed."
	}
	return message + " No automatic retry was attempted. Inspect any installation receipt before explicitly retrying."
}

// A header can be represented under differently cased map keys in test/custom
// transports. Count all values, not just Header.Get's first canonical match.
func releaseHeaderNumber(h http.Header, name string) (uint64, bool) {
	var value string
	count := 0
	for key, values := range h {
		if strings.EqualFold(key, name) {
			count += len(values)
			if len(values) == 1 {
				value = values[0]
			}
		}
	}
	if count != 1 || len(value) == 0 || len(value) > 64 {
		return 0, false
	}
	for _, c := range []byte(value) {
		if c < '0' || c > '9' {
			return 0, false
		}
	}
	v, err := strconv.ParseUint(value, 10, 64)
	return v, err == nil
}

func releaseRateLimit(status int, h http.Header, now time.Time) *ReleaseRateLimitError {
	if status != http.StatusForbidden && status != http.StatusTooManyRequests {
		return nil
	}
	remaining, remainingOK := releaseHeaderNumber(h, "X-RateLimit-Remaining")
	after, afterOK := releaseHeaderNumber(h, "Retry-After")
	afterOK = afterOK && after >= 1 && after <= 86400
	primary := remainingOK && remaining == 0
	if status == http.StatusForbidden && !primary && !afterOK {
		return nil
	}
	r := ReleaseRetry{HTTPStatus: status, Kind: "unspecified"}
	if primary {
		r.Kind = "primary"
	}
	if afterOK {
		r.RetryAfterSeconds = int64(after)
	}
	if epoch, ok := releaseHeaderNumber(h, "X-RateLimit-Reset"); ok && epoch > 0 && epoch <= 1<<63-1 {
		reset := time.Unix(int64(epoch), 0).UTC()
		if reset.After(now) && reset.Sub(now) <= 24*time.Hour {
			r.ResetAt = reset.Format(time.RFC3339)
		}
	}
	return &ReleaseRateLimitError{Retry: r}
}
