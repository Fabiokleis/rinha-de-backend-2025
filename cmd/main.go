package main

import (
	"flag"

	api "rinha/internal/api"
	db "rinha/internal/database"
	listener "rinha/internal/listener"
)

var routes = flag.Bool("routes", false, "Generate router documentation")

func main() {
	defer func() {
		listener.Stop()
		db.Disconnect()
	}()

	flag.Parse()
	db.Connect()
	listener.Listen()
	api.CreateRoutes(*routes)
}
