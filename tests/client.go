package main

import (
	"context"
	"fmt"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pb "saarwasserman.com/notifications/grpcgen/proto"
)

func main() {
	req := &pb.SendActivationEmailRequest{
		Recipient: "test1@test1.com",
		UserId: "1",
		Token: "aaaa",
	}

	var opts []grpc.DialOption

	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	conn, err := grpc.NewClient("localhost:8090", opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
		return
	}
	defer conn.Close()

	client := pb.NewEMailServiceClient(conn)
	res, err := client.SendActivationEmail(context.Background(), req)
	if err != nil {
		log.Fatal("couldn't greet", err.Error())
		return
	}

	fmt.Println(res)
}