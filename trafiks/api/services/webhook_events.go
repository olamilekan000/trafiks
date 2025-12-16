package services

// Webhook event type constants
const (
	// Upstream events
	EventUpstreamUnreachable = "upstream.unreachable"
	EventUpstreamTimeout     = "upstream.timeout"

	// Request events
	EventRequestFailed = "request.failed"

	// Cache events
	EventCacheHit  = "cache.hit"
	EventCacheMiss = "cache.miss"

	// Project events
	EventProjectActivated   = "project.activated"
	EventProjectDeactivated = "project.deactivated"

	// API Key events
	EventSecretKeyCreated     = "secretkey.created"
	EventSecretKeyDeactivated = "secretkey.deactivated"

	// Error rate events
	EventErrorRateHigh = "error_rate.high"
)

// AllWebhookEvents returns a map of all available webhook events
func AllWebhookEvents() map[string]bool {
	return map[string]bool{
		EventUpstreamUnreachable:  true,
		EventUpstreamTimeout:      true,
		EventRequestFailed:        true,
		EventCacheHit:             true,
		EventCacheMiss:            true,
		EventProjectActivated:     true,
		EventProjectDeactivated:   true,
		EventSecretKeyCreated:     true,
		EventSecretKeyDeactivated: true,
		EventErrorRateHigh:        true,
	}
}

// IsValidWebhookEvent checks if an event type is valid
func IsValidWebhookEvent(eventType string) bool {
	allEvents := AllWebhookEvents()
	_, exists := allEvents[eventType]
	return exists
}
