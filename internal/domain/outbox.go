package domain

type OutboxEvent struct {
	ID        string
	Key       string
	EventType string
	Payload   []byte
}
