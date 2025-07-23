package api

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	pay "rinha/internal/api/payments"
	db "rinha/internal/database"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/jackc/pgx/v5/pgconn"
)

func CreateRoutes(gendoc bool) *http.Server {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(render.SetContentType(render.ContentTypeJSON))

	var greeting string
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		err := db.Pgxpool.QueryRow(db.PgxCtx, "select 'Hello, world!'").Scan(&greeting)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			fmt.Println(pgErr.Message) // => syntax error at end of input
			fmt.Println(pgErr.Code)    // => 42601
		}

		w.Write([]byte(greeting))
	})

	pay.NewRouter(r, gendoc)

	server := &http.Server{Addr: ":9999", Handler: r}

	// start http server in non-blocking
	go func() {
		fmt.Println("api started :9999")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server ListenAndServe: %v", err)
		}
	}()
	return server
}
