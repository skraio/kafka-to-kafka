package main

import (
    "fmt"
    "os"
    "os/signal"
    "github.com/confluentinc/confluent-kafka-go/kafka"
)

func main() {
    c, err := kafka.NewConsumer(&kafka.ConfigMap{
        "bootstrap.servers": "kafka:29092", // for internal communication (via Docker)
        // "bootstrap.servers": "localhost:9092", // for external communication (via command line go run ./...)
        "group.id":          "myGroup",
        "auto.offset.reset": "earliest",
    })
    if err != nil {
        panic(err)
    }

    c.SubscribeTopics([]string{"incoming-transactions"}, nil)

    sigchan := make(chan os.Signal, 1)
    signal.Notify(sigchan, os.Interrupt)

    run := true

    for run {
        select {
        case sig := <-sigchan:
            fmt.Printf("Caught signal %v: terminating\n", sig)
            run = false
        default:
            ev := c.Poll(100)
            switch e := ev.(type) {
            case *kafka.Message:
                fmt.Printf("%% Message on %s:\n%s\n", e.TopicPartition, string(e.Value))
                _, err := c.CommitMessage(e)
                if err != nil {
                    fmt.Printf("%% Failed to commit message: %v\n", err)
                }
            case kafka.Error:
                fmt.Fprintf(os.Stderr, "%% Error: %v\n", e)
                run = false
            }
        }
    }

    c.Close()
}
