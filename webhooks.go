package forumulate

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// WebhookInput is one delivery to verify with VerifyWebhookSignature.
type WebhookInput struct {
	// Secret is the endpoint's signing secret.
	Secret string
	// Body is the raw request body, exactly as received (verify BEFORE decoding).
	Body []byte
	// Signature is the whole X-Forumulate-Signature header.
	Signature string
	// Timestamp is the X-Forumulate-Timestamp header (unix seconds).
	Timestamp string
	// Tolerance is how old (or far in the future) a delivery may be. Zero
	// means the default of 5 minutes; a negative value skips the check.
	Tolerance time.Duration
	// AcceptLegacy also accepts the replayable body-only "sha256=" signature.
	AcceptLegacy bool
	// Now overrides the clock (tests).
	Now func() time.Time
}

var webhookEntry = regexp.MustCompile(`(v1,)?sha256=([0-9a-fA-F]+)`)

// VerifyWebhookSignature reports whether a delivery was signed by Forumulate.
//
// The signature header is a comma-separated LIST: during a secret rotation
// both secrets sign, and the legacy body-only "sha256=" rides along while an
// endpoint's signatureVersion is "both". A delivery is genuine if ANY entry
// matches. "v1" signs "<timestamp>.<raw body>", which is what makes a
// captured delivery un-replayable, so the legacy entry is accepted only when
// AcceptLegacy is set.
func VerifyWebhookSignature(in WebhookInput) bool {
	if in.Secret == "" || in.Signature == "" {
		return false
	}
	ts, err := strconv.ParseInt(strings.TrimSpace(in.Timestamp), 10, 64)
	if err != nil || ts <= 0 {
		return false
	}
	tol := in.Tolerance
	if tol == 0 {
		tol = 5 * time.Minute
	}
	if tol > 0 {
		now := time.Now()
		if in.Now != nil {
			now = in.Now()
		}
		d := now.Unix() - ts
		if d < 0 {
			d = -d
		}
		if time.Duration(d)*time.Second > tol {
			return false
		}
	}
	sign := func(msg []byte) string {
		m := hmac.New(sha256.New, []byte(in.Secret))
		m.Write(msg)
		return hex.EncodeToString(m.Sum(nil))
	}
	v1 := sign(append([]byte(strconv.FormatInt(ts, 10)+"."), in.Body...))
	legacy := ""
	if in.AcceptLegacy {
		legacy = sign(in.Body)
	}
	ok := false
	for _, m := range webhookEntry.FindAllStringSubmatch(in.Signature, -1) {
		offered := []byte(strings.ToLower(m[2]))
		if m[1] != "" {
			ok = hmac.Equal(offered, []byte(v1)) || ok
		} else if legacy != "" {
			ok = hmac.Equal(offered, []byte(legacy)) || ok
		}
	}
	return ok
}
