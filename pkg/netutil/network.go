package netutil

import (
	"errors"
	"net"
	"strings"
)

// IsTransientNetworkError reports whether err is a network failure that is
// likely to succeed on retry (DNS temporarily unavailable, timeouts, resets).
func IsTransientNetworkError(err error) bool {
	if err == nil {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) && (netErr.Timeout() || netErr.Temporary()) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "temporary failure in name resolution") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "actively refused") ||
		strings.Contains(msg, "no connection could be made")
}