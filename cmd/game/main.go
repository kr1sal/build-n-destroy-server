package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"kr1sal.com/build-n-destroy/internal/game"
)

func main() {
	rootCtx := context.Background()
	ctx, stop := signal.NotifyContext(rootCtx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Println("Server started!")

	go game.Start(ctx)

	<-ctx.Done()
	fmt.Println("Server stopped!")
}