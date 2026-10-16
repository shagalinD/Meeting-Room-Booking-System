package kafka

import (
	"fmt"
	"os"
	"strings"
)

const (
	defaultBroker        = "localhost:9092"
	defaultTopic         = "booking.events"
	defaultConsumerGroup = "booking-event-logger"
)

type Config struct {
	Brokers       []string
	Topic         string
	ConsumerGroup string
}

func ConfigFromEnv() (Config, error) {
	rawBrokers := strings.TrimSpace(os.Getenv("KAFKA_BROKERS"))
	if rawBrokers == "" {
		rawBrokers = defaultBroker
	}

	brokers := make([]string, 0)
	for _, broker := range strings.Split(rawBrokers, ",") {
		broker = strings.TrimSpace(broker)
		if broker == "" {
			return Config{}, fmt.Errorf("KAFKA_BROKERS contains an empty broker address")
		}
		brokers = append(brokers, broker)
	}

	topic := strings.TrimSpace(os.Getenv("KAFKA_TOPIC"))
	if topic == "" {
		topic = defaultTopic
	}
	group := strings.TrimSpace(os.Getenv("KAFKA_CONSUMER_GROUP"))
	if group == "" {
		group = defaultConsumerGroup
	}

	return Config{Brokers: brokers, Topic: topic, ConsumerGroup: group}, nil
}
