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
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"saarwasserman.com/notifications/internal/jsonlog"
	"saarwasserman.com/notifications/internal/vcs"

	pb "saarwasserman.com/notifications/grpcgen/proto"
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
	smtp struct {
		host     string
		port     int
		username string
		password string
		sender   string
	}
	cors struct {
		trustedOrigins []string
	}
}

type application struct {
	pb.UnimplementedEMailServiceServer
	config config
	logger *jsonlog.Logger
	queue *kafka.Writer
	wg     sync.WaitGroup
}

func main() {
	var cfg config

	// server
	flag.IntVar(&cfg.port, "port", 4000, "GRPC Server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment(development|staging|production)")

	// kafka
	flag.StringVar(&cfg.smtp.host, "kafka-host", "localhost", "Kafka host")
	flag.IntVar(&cfg.smtp.port, "kafka-port", 9092, "Kafka port")

	// limiter
	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", 2, "Rate limiter maximum requests per second")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 4, "Rate limiter maximum burst")
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", true, "Enable rate limiter")

	// mailer
	flag.StringVar(&cfg.smtp.host, "smtp-host", "sandbox.smtp.mailtrap.io", "SMTP host")
	flag.IntVar(&cfg.smtp.port, "smtp-port", 2525, "SMTP port")
	flag.StringVar(&cfg.smtp.username, "smtp-username", os.Getenv("GREENLIGHT_SMTP_USERNAME"), "SMTP username")
	flag.StringVar(&cfg.smtp.password, "smtp-password", os.Getenv("GREENLIGHT_SMTP_PASSWORD"), "SMTP password")
	flag.StringVar(&cfg.smtp.sender, "smtp-sender", "Greenlight <no-reply@greenlight.saarw.net>", "SMTP sender")

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
			Addr: 	kafka.TCP("localhost:9092"),
    		// NOTE: When Topic is not defined here, each Message must define it instead.
			Balancer: &kafka.LeastBytes{},
		},
	}

	defer app.queue.Close()

	listener, err := net.Listen("tcp", ":8090")
	if err != nil {
		log.Fatalf("cannot create listener %s", err)
		return
	}


	serviceRegistrar := grpc.NewServer()

	pb.RegisterEMailServiceServer(serviceRegistrar, app)

	err = serviceRegistrar.Serve(listener)
	if err != nil {
		log.Fatalf("cannot serve %s", err)
		return
	}
}
		

func (app *application) SendActivationEmail(ctx context.Context, req *pb.SendActivationEmailRequest) (*pb.SendActivationEmailResponse, error) {

	buff, err := proto.Marshal(&pb.ActivationEmailRequest{
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

	return &pb.SendActivationEmailResponse{}, nil
}
