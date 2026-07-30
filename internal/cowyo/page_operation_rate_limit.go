package cowyo

import (
	"crypto/sha256"
	"net/http"
	"time"
)

const (
	pageOperationClientRequests = 20
	pageOperationClientPeriod   = time.Minute
	pageOperationClientBurst    = 8

	pageOperationTargetRequests = 60
	pageOperationTargetPeriod   = time.Hour
	pageOperationTargetBurst    = 10

	pageCredentialClientRequests = 6
	pageCredentialClientPeriod   = 10 * time.Minute
	pageCredentialClientBurst    = 3

	pageCredentialTargetRequests = 10
	pageCredentialTargetPeriod   = time.Hour
	pageCredentialTargetBurst    = 3
)

var (
	pageOperationClientLimiter = newPostRateLimiter(
		pageOperationClientRequests,
		pageOperationClientPeriod,
		pageOperationClientBurst,
	)
	pageOperationTargetLimiter = newPostRateLimiter(
		pageOperationTargetRequests,
		pageOperationTargetPeriod,
		pageOperationTargetBurst,
	)
	pageCredentialClientLimiter = newPostRateLimiter(
		pageCredentialClientRequests,
		pageCredentialClientPeriod,
		pageCredentialClientBurst,
	)
	pageCredentialTargetLimiter = newPostRateLimiter(
		pageCredentialTargetRequests,
		pageCredentialTargetPeriod,
		pageCredentialTargetBurst,
	)
)

func allowPageOperation(
	r *http.Request,
	title string,
	operation string,
) (bool, time.Duration) {
	client := postClientKey(r)
	target := pageOperationRateKey(title)

	if allowed, retryAfter := pageOperationClientLimiter.Allow(client); !allowed {
		return false, retryAfter
	}
	if allowed, retryAfter := pageOperationTargetLimiter.Allow(target); !allowed {
		return false, retryAfter
	}

	if operation != operationLock && operation != operationUnlock {
		return true, 0
	}
	if allowed, retryAfter := pageCredentialClientLimiter.Allow(
		client + "\x00" + target,
	); !allowed {
		return false, retryAfter
	}
	return pageCredentialTargetLimiter.Allow(target)
}

func pageOperationRateKey(title string) string {
	hash := sha256.Sum256([]byte(title))
	return string(hash[:])
}
