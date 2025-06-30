package logger

import (
	"io"
	"os"

	"github.com/IBM/sarama"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

// New creates a JSON structured logger at the specified level.
type KafkaHook struct {
	producer sarama.SyncProducer
	topic    string
}

type RabbitMQHook struct {
	ch    *amqp.Channel
	queue string
}

// Fire writes the log entry to Kafka.
func (h *KafkaHook) Fire(e *logrus.Entry) error {
	b, err := e.Logger.Formatter.Format(e)
	if err != nil {
		return err
	}
	msg := &sarama.ProducerMessage{Topic: h.topic, Value: sarama.ByteEncoder(b)}
	_, _, err = h.producer.SendMessage(msg)
	return err
}

func (h *KafkaHook) Levels() []logrus.Level { return logrus.AllLevels }

func (h *RabbitMQHook) Fire(e *logrus.Entry) error {
	b, err := e.Logger.Formatter.Format(e)
	if err != nil {
		return err
	}
	return h.ch.Publish("", h.queue, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        b,
	})
}

func (h *RabbitMQHook) Levels() []logrus.Level { return logrus.AllLevels }

// New creates a JSON structured logger at the specified level. Logs are sent
// to Kafka only when env is "production" and brokers are configured.
// New creates a JSON structured logger at the specified level.
// MQ driver can be "kafka" or "rabbitmq". File logging is enabled when env is
// not "production" and filePath is provided (or defaulted).
func New(level, env, driver string, brokers []string, topic, rabbitURL, rabbitQueue, filePath string) *logrus.Logger {
	log := logrus.New()
	log.SetOutput(os.Stdout)
	log.Formatter = &logrus.JSONFormatter{}

	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		lvl = logrus.InfoLevel
	}
	log.SetLevel(lvl)

	// Non-production: enable file logging only
	if env != "production" {
		if filePath == "" {
			filePath = "app.log"
		}
		f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err == nil {
			log.SetOutput(io.MultiWriter(os.Stdout, f))
		} else {
			log.WithError(err).Warn("cannot open log file")
		}
		return log // 💥 Return early: skip MQ setup
	}

	// Only in production: setup message queue hooks
	switch driver {
	case "kafka":
		if len(brokers) > 0 {
			cfg := sarama.NewConfig()
			cfg.Producer.Return.Successes = true
			producer, err := sarama.NewSyncProducer(brokers, cfg)
			if err == nil {
				if topic == "" {
					topic = "logging"
				}
				log.AddHook(&KafkaHook{producer: producer, topic: topic})
			} else {
				log.WithError(err).Warn("failed to initialize kafka producer")
			}
		}
	case "rabbitmq":
		if rabbitURL != "" {
			conn, err := amqp.Dial(rabbitURL)
			if err != nil {
				log.WithError(err).Warn("failed to connect rabbitmq")
				break
			}
			ch, err := conn.Channel()
			if err != nil {
				log.WithError(err).Warn("failed to open rabbitmq channel")
				break
			}
			if rabbitQueue == "" {
				rabbitQueue = "logging"
			}
			_, err = ch.QueueDeclare(rabbitQueue, true, false, false, false, nil)
			if err != nil {
				log.WithError(err).Warn("failed to declare rabbitmq queue")
				break
			}
			log.AddHook(&RabbitMQHook{ch: ch, queue: rabbitQueue})
		}
	}

	return log
}
