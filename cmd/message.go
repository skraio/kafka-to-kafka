package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

type Message struct {
	ID      int     `json:"id"`
	Login   string  `json:"login"`
	Payment float64 `json:"payment"`
	Status  string  `json:"status"`
}

func (app *application) processMessage(kafkaMessage *kafka.Message) ([]byte, string, error) {
	var message Message
	if err := json.Unmarshal(kafkaMessage.Value, &message); err != nil {
		return nil, "", err
	}

	topic, err := app.filter(message)
	if err != nil {
		return nil, "", err
	}

	transformedMessage := app.transform(message)
	if err := app.deduplicate(transformedMessage); err != nil {
		return nil, "", err
	}

	result, err := json.Marshal(transformedMessage)
	if err != nil {
		return nil, "", err
	}

	return result, topic, nil
}

func (app *application) filter(message Message) (string, error) {
	filterValue1 := app.config.kafkaMessage.config.Filtering.Value1
	filterValue2 := app.config.kafkaMessage.config.Filtering.Value2

	switch message.Status {
	case filterValue1:
		return app.config.kafka.producer.topicStandard, nil
	case filterValue2:
		return app.config.kafka.producer.topicPrivileged, nil
	}

    err := fmt.Errorf("message does not meet filtering criteria")
	return "", err
}

func (app *application) transform(message Message) map[string]interface{} {
	result := make(map[string]interface{})
	for _, key := range app.config.kafkaMessage.config.Transformation {
		switch key {
		case "id":
			result["id"] = message.ID
		case "login":
			result["login"] = message.Login
		case "payment":
			result["payment"] = message.Payment
		}
	}
	return result
}

func (app *application) deduplicate(message map[string]interface{}) error {
	ctx := context.Background()
	key := fmt.Sprintf("%s", message[app.config.kafkaMessage.config.Deduplication.Key])
	duration := time.Duration(app.config.kafkaMessage.config.Deduplication.TimeSpanSeconds) * time.Second

	result, err := app.redisClient.SetNX(ctx, key, true, duration).Result()
	if err != nil {
		return err
	}

	if !result {
		return fmt.Errorf("duplicate message detected")
	}

	return nil
}
