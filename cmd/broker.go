package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

func (app *application) readWriteMessages() error {
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt)

	handleMessage := func(e *kafka.Message) {
		msg, topic, err := app.processMessage(e)
		if err != nil {
			fmt.Printf("Skipping message: %s, error: %v\n", string(e.Value), err)
			return
		}

		deliveryChan := make(chan kafka.Event, 1)
		err = app.producer.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
			Value:          msg,
		}, deliveryChan)

		if err != nil {
			fmt.Printf("Failed to produce message: %v, error: %v\n", msg, err)
			close(deliveryChan)
			return
		}

		ev := <-deliveryChan
		m := ev.(*kafka.Message)
		if m.TopicPartition.Error != nil {
			fmt.Printf("Delivery failed: %v\n", m.TopicPartition.Error)
		}
        close(deliveryChan)
	}

	run := true
	for run {
		select {
		case sig := <-sigchan:
			fmt.Printf("Caught signal %v: terminating\n", sig)
			run = false
			cancel()
		default:
			ev := app.consumer.Poll(100)
			switch e := ev.(type) {
			case *kafka.Message:
				handleMessage(e)
			case kafka.Error:
				fmt.Fprintf(os.Stderr, "%% Error: %v\n", e)
				run = false
			}
		}
	}

	return nil
}
