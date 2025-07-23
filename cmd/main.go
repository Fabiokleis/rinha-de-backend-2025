package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"rinha/internal/api"
	db "rinha/internal/database"
	listener "rinha/internal/listener"
)

var gendoc = flag.Bool("routes", false, "Generate router documentation")

func main() {

	flag.Parse()
	db.Connect()
	listener.Listen()
	server := api.CreateRoutes(*gendoc)

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
	<-sc
	fmt.Println("shutdown amigo...")

	listener.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond) // Set a timeout for shutdown
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		fmt.Println(err.Error())
		fmt.Println("failed to stop api http server")
	}
	fmt.Println("api stopped")
	db.Disconnect()
}
