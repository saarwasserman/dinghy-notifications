package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
	pb "saarwasserman.com/notifications/grpcgen/proto"
	"saarwasserman.com/notifications/internal/jsonlog"
	"saarwasserman.com/notifications/internal/mailer"
)


type config struct {
	env  string
	// limiter struct {
	// 	rps     float64
	// 	burst   int
	// 	enabled bool
	// }
	smtp struct {
		host     string
		port     int
		username string
		password string
		sender   string
	}
	kafka struct {
		host string
		port int
		topic string
	}
}


type application struct {
	config config
	logger *jsonlog.Logger
	consumer *kafka.Reader
	mailer mailer.Mailer
	wg     sync.WaitGroup
}


func (app *application) background(fn func()) {

	app.wg.Add(1)

	go func() {
		defer app.wg.Done()

		defer func() {
			if err := recover(); err != nil {
				app.logger.PrintError(fmt.Errorf("%s", err), nil)
			}
		}()

		fn()
	}()
}


func main() {
	var cfg config

	flag.StringVar(&cfg.env, "env", "development", "Environment(development|staging|production)")

	// mailer
	flag.StringVar(&cfg.smtp.host, "smtp-host", "sandbox.smtp.mailtrap.io", "SMTP host")
	flag.IntVar(&cfg.smtp.port, "smtp-port", 2525, "SMTP port")
	flag.StringVar(&cfg.smtp.username, "smtp-username", os.Getenv("GREENLIGHT_SMTP_USERNAME"), "SMTP username")
	flag.StringVar(&cfg.smtp.password, "smtp-password", os.Getenv("GREENLIGHT_SMTP_PASSWORD"), "SMTP password")
	flag.StringVar(&cfg.smtp.sender, "smtp-sender", "Greenlight <no-reply@greenlight.saarw.net>", "SMTP sender")

	// kafka
	flag.StringVar(&cfg.kafka.host, "kafka-host", "localhost", "Kafka host")
	flag.IntVar(&cfg.kafka.port, "kafka-port", 9092, "Kafka port")
	flag.StringVar(&cfg.kafka.topic, "kafka-topic", "general-email", "Kafka topic")


	flag.Parse()


	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)


	app := &application{
		config: cfg,
		logger: logger,
		consumer: kafka.NewReader(kafka.ReaderConfig{
			Brokers:   []string{"localhost:9092"},
			Topic:     "general-email",
			Partition: 0,
			MaxBytes:  10e6, // 10MB
		}),
		mailer: mailer.New(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender),
	}


	for {
		m, err := app.consumer.ReadMessage(context.Background())

		emailData := &pb.ActivationEmailRequest{}
		proto.Unmarshal(m.Value, emailData)
		if err != nil {
			break
		}

		app.background(func() {
			data := map[string]any{
				"activationToken": emailData.Token,
				"userID":          emailData.UserId,
			}

			err = app.mailer.Send(emailData.Recipient, "user_welcome.tmpl", data)
			if err != nil {
				app.logger.PrintError(err, nil)
			}
		})
	}

	if err := app.consumer.Close(); err != nil {
		log.Fatal("failed to close reader:", err)
	}
}
