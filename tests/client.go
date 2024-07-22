package main

import (
	"context"
	"fmt"
	"log"

	"github.com/saarwasserman/notifications/protogen/notifications"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	req := &notifications.SendActivationEmailRequest{
		Recipient: "test1@test1.com",
		UserId: "1",
		Token: "aaaa",
	}

	var opts []grpc.DialOption

	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	conn, err := grpc.NewClient("localhost:40010", opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
		return
	}
	defer conn.Close()

	client := notifications.NewEMailServiceClient(conn)
	res, err := client.SendActivationEmail(context.Background(), req)
	if err != nil {
		log.Fatal("couldn't greet", err.Error())
		return
	}

	fmt.Println(res)
}