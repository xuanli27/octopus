package relay

import "errors"

const (
	CodeRelayModelNotSupported     = "relay.model_not_supported"
	CodeRelayModelNotFound         = "relay.model_not_found"
	CodeRelayNoAvailableChannel    = "relay.no_available_channel"
	CodeRelayChannelDisabled       = "relay.channel_disabled"
	CodeRelayNoAvailableKey        = "relay.no_available_key"
	CodeRelayUpstreamFailed        = "relay.upstream_failed"
	CodeRelayTimeout               = "relay.timeout"
	CodeRelayCircuitBreakerTripped = "relay.circuit_breaker_tripped"
)

// ErrEmptyUpstreamResponse marks a 200 non-stream body that carries no usable
// completion content (no choices, embeddings, error, or usage). Relay treats it
// as a retryable upstream failure so the client is not left with a blank session.
// See GitHub issue #100.
var ErrEmptyUpstreamResponse = errors.New("upstream returned empty response body")
