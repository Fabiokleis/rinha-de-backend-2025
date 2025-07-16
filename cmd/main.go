package main

import (
	"flag"

	api "rinha/internal/api"
	db "rinha/internal/database"
)

var routes = flag.Bool("routes", false, "Generate router documentation")

func main() {
	defer db.Disconnect()
	flag.Parse()
	db.Connect()
	api.CreateRoutes(*routes)
}
