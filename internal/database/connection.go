package db

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

var PgxCtx context.Context

var Pgxpool *pgxpool.Pool

func Connect() {
	PgxCtx = context.Background()
	pool, err := pgxpool.New(PgxCtx, os.Getenv("DB_CONNECTION_STRING"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create connection pool: %v\n", err)
		os.Exit(1)
	}
	Pgxpool = pool
	pingErr := Pgxpool.Ping(PgxCtx)
	if pingErr != nil {
		fmt.Fprintf(os.Stderr, "Unable to acquire connection pool: %v\n", pingErr)
		os.Exit(1)
	}

	fmt.Println("postgresql connected")
}

func Disconnect() {
	if Pgxpool != nil {
		Pgxpool.Close()
		fmt.Println("postgresql disconnected")
		return
	}
	fmt.Fprintln(os.Stderr, "Trying to close nil connection pool")
	os.Exit(1)
}
