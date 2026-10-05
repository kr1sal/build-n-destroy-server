package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"kr1sal.com/build-n-destroy/internal/game"
	"kr1sal.com/build-n-destroy/internal/pb"
	"kr1sal.com/build-n-destroy/internal/transport"
)

func main() {
	rootCtx := context.Background()
	ctx, stop := signal.NotifyContext(rootCtx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	listener, err := net.Listen("tcp", ":50051")
	defer listener.Close()
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	defer grpcServer.GracefulStop()
	room := game.NewRoom()
	pb.RegisterGameServiceServer(grpcServer, transport.NewGameServer(room))

	fmt.Println("Server started!")

	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
		stop()
	}()

	go room.Run(ctx)

	<-ctx.Done()
	fmt.Println("Server stopped!")
}
