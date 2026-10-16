package kafka

import (
	"reflect"
	"testing"
)

func TestConfigFromEnvDefaults(t *testing.T) {
	t.Setenv("KAFKA_BROKERS", "")
	t.Setenv("KAFKA_TOPIC", "")
	t.Setenv("KAFKA_CONSUMER_GROUP", "")

	config, err := ConfigFromEnv()
	if err != nil {
		t.Fatalf("ConfigFromEnv() error = %v", err)
	}
	if !reflect.DeepEqual(config.Brokers, []string{"localhost:9092"}) ||
		config.Topic != "booking.events" ||
		config.ConsumerGroup != "booking-event-logger" {
		t.Fatalf("ConfigFromEnv() = %+v", config)
	}
}

func TestConfigFromEnvParsesBrokers(t *testing.T) {
	t.Setenv("KAFKA_BROKERS", "kafka:9092, kafka-backup:9092")
	t.Setenv("KAFKA_TOPIC", "custom.events")
	t.Setenv("KAFKA_CONSUMER_GROUP", "custom-consumer")

	config, err := ConfigFromEnv()
	if err != nil {
		t.Fatalf("ConfigFromEnv() error = %v", err)
	}
	if !reflect.DeepEqual(config.Brokers, []string{"kafka:9092", "kafka-backup:9092"}) ||
		config.Topic != "custom.events" ||
		config.ConsumerGroup != "custom-consumer" {
		t.Fatalf("ConfigFromEnv() = %+v", config)
	}
}

func TestConfigFromEnvRejectsEmptyBrokerEntry(t *testing.T) {
	t.Setenv("KAFKA_BROKERS", "kafka:9092,")

	if _, err := ConfigFromEnv(); err == nil {
		t.Fatal("ConfigFromEnv() expected an error")
	}
}
