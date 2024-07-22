package main

import (
	"context"
	"expvar"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/saarwasserman/notifications/internal/jsonlog"
	"github.com/saarwasserman/notifications/internal/vcs"
	"github.com/segmentio/kafka-go"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	"github.com/saarwasserman/notifications/protogen/notifications"
)

var (
	version = vcs.Version()
)

type config struct {
	port int
	env  string
	limiter struct {
		rps     float64
		burst   int
		enabled bool
	}
	kafka struct {
		host string
		port int
		topic string
	}
	cors struct {
		trustedOrigins []string
	}
}

type application struct {
	notifications.UnimplementedEMailServiceServer
	config config
	logger *jsonlog.Logger
	queue *kafka.Writer
}


func main() {
	var cfg config

	// server
	flag.IntVar(&cfg.port, "port", 40010, "GRPC Server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment(development|staging|production)")

	// kafka
	flag.StringVar(&cfg.kafka.host, "kafka-host", "localhost", "Kafka host")
	flag.IntVar(&cfg.kafka.port, "kafka-port", 9092, "Kafka port")
	flag.StringVar(&cfg.kafka.topic, "kafka-topic", "general-email", "Kafka topic")

	// limiter
	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", 2, "Rate limiter maximum requests per second")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 4, "Rate limiter maximum burst")
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", true, "Enable rate limiter")


	// cors
	flag.Func("cors-trusted-origins", "Trusted CORS Origins (space separated)", func(val string) error {
		cfg.cors.trustedOrigins = strings.Fields(val)
		return nil
	})

	displayVersion := flag.Bool("version", false, "Display version and exit")

	flag.Parse()

	if *displayVersion {
		fmt.Printf("Version:\t%s\n", version)
		os.Exit(0)
	}

	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)

	expvar.NewString("version").Set(version)

	expvar.Publish("goroutins", expvar.Func(func() any {
		return runtime.NumGoroutine()
	}))

	expvar.Publish("timestamp", expvar.Func(func() any {
		return time.Now().Unix()
	}))

	app := &application{
		config: cfg,
		logger: logger,
		queue: &kafka.Writer{
			Addr: 	kafka.TCP(fmt.Sprintf("%s:%d", cfg.kafka.host, cfg.kafka.port)),
    		// NOTE: When Topic is not defined here, each Message must define it instead.
			Balancer: &kafka.LeastBytes{},
		},
	}

	defer app.queue.Close()

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", app.config.port))
	if err != nil {
		log.Fatalf("cannot create listener %s", err)
		return
	}

	serviceRegistrar := grpc.NewServer()

	notifications.RegisterEMailServiceServer(serviceRegistrar, app)

	err = serviceRegistrar.Serve(listener)
	if err != nil {
		log.Fatalf("cannot serve %s", err)
		return
	}
}


func (app *application) SendActivationEmail(ctx context.Context, req *notifications.SendActivationEmailRequest) (*notifications.SendActivationEmailResponse, error) {

	buff, err := proto.Marshal(&notifications.ActivationEmailRequest{
		Recipient: req.Recipient,
		UserId: req.UserId,
		Token: req.Token,
		TemplateFile: "user_welcome.tmpl",
	})
	if err != nil {
		return nil, err
	}

	err = app.queue.WriteMessages(context.Background(),
		kafka.Message{
			Topic: "general-email",
			Key:   []byte(req.UserId + "activationemail"),
			Value: buff,
		},
	)

	if err != nil {
		log.Fatal("failed to write messages:", err)
		return nil, err
	}

	return &notifications.SendActivationEmailResponse{}, nil
}
