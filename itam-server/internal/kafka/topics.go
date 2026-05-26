// Package kafka owns topic names, producer, and consumer-group primitives.
package kafka

const (
	TopicScanFull  = "agent.scan.full"
	TopicScanDiff  = "agent.scan.diff"
	TopicHeartbeat = "agent.heartbeat"
	TopicAlert     = "agent.alert"

	GroupPGWriter    = "pg-writer"
	GroupGraphWriter = "graph-writer"
	GroupAnomaly     = "anomaly-engine"
	GroupAlerter     = "alert-forwarder"
)
