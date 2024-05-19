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

    topic := app.filter(message)
    transformedMessage := app.transform(message)

    if !app.deduplicate(transformedMessage) {
        return []byte{}, "", nil
    }

	result, err := json.Marshal(transformedMessage)
	if err != nil {
		fmt.Printf("%% Error marshalling transformed message: %v\n", err)
		return nil, "", err
	}

	return result, topic, nil
}

func (app *application) filter(message Message) string {
	filterValue1 := app.config.kafkaMessage.config.Filtering.Value1
	filterValue2 := app.config.kafkaMessage.config.Filtering.Value2

	switch message.Status {
	case filterValue1:
		return app.config.kafka.producer.topicStandard
	case filterValue2:
		return app.config.kafka.producer.topicPrivileged
	}
    return ""
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

func (app *application) deduplicate(message map[string]interface{}) bool {
	ctx := context.Background()
	key := fmt.Sprintf("%s", message[app.config.kafkaMessage.config.Deduplication.Key])
	duration := time.Duration(app.config.kafkaMessage.config.Deduplication.TimeSpanSeconds) * time.Second

	exists, err := app.redisClient.SetNX(ctx, key, true, duration).Result()
	if err != nil {
		fmt.Printf("%% Error accessing Redis: %v\n", err)
		return false
	}

	return exists
}
