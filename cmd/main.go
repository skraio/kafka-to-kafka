package main

import (
	"context"
	"encoding/json"
	"flag"
	"log/slog"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/redis/go-redis/v9"
)

type config struct {
	kafka struct {
		bootstrap string
		groupId   string
		offset    string

		consumer struct {
			topicIncoming string
		}
		producer struct {
			topicStandard   string
			topicPrivileged string
		}
	}
	redis struct {
		addr     string
		password string
		db       int
	}
	kafkaMessage struct {
		configFile string
		config     messageConfig
	}
}

type messageConfig struct {
	Filtering struct {
		Key    string `json:"key"`
		Value1 string `json:"value1"`
		Value2 string `json:"value2"`
	}
	Transformation []string `json:"transformation"`
	Deduplication  struct {
		Key             string `json:"key"`
		TimeSpanSeconds int    `json:"time_span_seconds"`
	}
}

type application struct {
	config      config
	logger      *slog.Logger
	consumer    *kafka.Consumer
	producer    *kafka.Producer
	redisClient *redis.Client
	wg          sync.WaitGroup
}

func main() {
	var cfg config
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	flag.StringVar(&cfg.redis.addr, "redis-addr", os.Getenv("REDIS_ADDR"), "Redis Addr")
	flag.StringVar(&cfg.redis.password, "redis-password", os.Getenv("REDIS_PASSWORD"), "Redis Password")
	redisDB, err := strconv.Atoi(os.Getenv("REDIS_DB"))
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	flag.IntVar(&cfg.redis.db, "redis-db", redisDB, "Redis DB")

	flag.StringVar(&cfg.kafka.bootstrap, "bootstrap", os.Getenv("BOOTSTRAP_SERVERS"), "")
	flag.StringVar(&cfg.kafka.groupId, "groupID", os.Getenv("GROUP_ID"), "")
	flag.StringVar(&cfg.kafka.offset, "offset", os.Getenv("OFFSET_RESET"), "")
	flag.StringVar(&cfg.kafka.consumer.topicIncoming, "consumer-topic", os.Getenv("CONSUMER_TOPIC"), "")
	flag.StringVar(&cfg.kafka.producer.topicStandard, "producer-topic-std", os.Getenv("CONSUMER_TOPIC_STD"), "")
	flag.StringVar(&cfg.kafka.producer.topicPrivileged, "producer-topic-priv", os.Getenv("CONSUMER_TOPIC_PRIV"), "")

	flag.StringVar(&cfg.kafkaMessage.configFile, "config-filename", os.Getenv("CONFIG_FILENAME"), "")

	flag.Parse()

	consumer, err := createConsumer(cfg)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	defer consumer.Close()
	logger.Info("Consumer created")

	producer, err := createProduce(cfg)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	defer producer.Close()
	logger.Info("Producer created")

	messageConfig, err := loadMessageConfig(cfg)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	cfg.kafkaMessage.config = messageConfig
	logger.Info("config.json loaded")

	redisClient, err := openRD(cfg)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	defer redisClient.Close()
	logger.Info("redis connetion pool established")

	app := &application{
		config:      cfg,
		logger:      logger,
		consumer:    consumer,
		producer:    producer,
		redisClient: redisClient,
	}

	err = app.readWriteMessages()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}

func createConsumer(cfg config) (*kafka.Consumer, error) {
	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": cfg.kafka.bootstrap,
		"group.id":          cfg.kafka.groupId,
		"auto.offset.reset": cfg.kafka.offset,
	})

	if err != nil {
		return nil, err
	}

	c.SubscribeTopics([]string{cfg.kafka.consumer.topicIncoming}, nil)

	return c, nil
}

func createProduce(cfg config) (*kafka.Producer, error) {
	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": cfg.kafka.bootstrap,
	})
	if err != nil {
		return nil, err
	}

	return p, nil
}

func loadMessageConfig(cfg config) (messageConfig, error) {
	var config messageConfig
	file, err := os.ReadFile(cfg.kafkaMessage.configFile)
	if err != nil {
		return config, err
	}
	err = json.Unmarshal(file, &config)
	return config, err
}

func openRD(cfg config) (*redis.Client, error) {
	rd := redis.NewClient(&redis.Options{
		Addr:     cfg.redis.addr,
		Password: cfg.redis.password,
		DB:       cfg.redis.db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := rd.Ping(ctx).Result()
	if err != nil {
		rd.Close()
		return nil, err
	}

	return rd, nil
}
