package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/docgen"
	"github.com/go-chi/render"

	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var routes = flag.Bool("routes", false, "Generate router documentation")

type Response struct {
	HTTPStatusCode int    `json:"-"`      // http response status code
	StatusText     string `json:"status"` // user-level status message

}

func (resp *Response) Render(w http.ResponseWriter, r *http.Request) error {
	render.Status(r, resp.HTTPStatusCode)
	return nil
}

func SuccessCreated() render.Renderer {
	return &Response{
		HTTPStatusCode: http.StatusCreated,
		StatusText:     "created",
	}
}

func ErrRender(err error) render.Renderer {
	return &Response{
		HTTPStatusCode: http.StatusUnprocessableEntity,
		StatusText:     "Error rendering response.",
	}
}

func ErrNotFound() render.Renderer {
	return &Response{
		HTTPStatusCode: http.StatusNotFound,
		StatusText:     "payment not found.",
	}
}

func ErrInvalidRequest(message string) render.Renderer {
	return &Response{
		HTTPStatusCode: http.StatusBadRequest,
		StatusText:     message,
	}
}

func ErrServerInternal() render.Renderer {
	return &Response{
		HTTPStatusCode: http.StatusInternalServerError,
		StatusText:     "unknown",
	}
}

type Payment struct {
	CorrelationId string  `json:"correlationId"`
	Amount        float32 `json:"amount"`
}

type PaymentRequest struct {
	*Payment
}

type PaymentDetailsRequest struct {
	Id string
}

func (pay *PaymentRequest) Bind(r *http.Request) error {
	if pay.Payment == nil {
		return errors.New("missing required payment fields.")
	}
	return nil
}

func (pay *PaymentDetailsRequest) Bind(r *http.Request) error {
	if pay == nil {
		return errors.New("missing id.")
	}
	return nil
}

type PaymentResponse struct {
	*Payment
	RequestedAt time.Time `json:"requested_at"`
}

func (pr *PaymentResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

type PaymentHandler struct {
	ctx context.Context
	db  *pgxpool.Pool
}

// POST /payments
// {
//     "correlationId": "4a7901b8-7d26-4d9d-aa19-4dc1c7cf60b3",
//     "amount": 19.90
// }

// HTTP 2XX
// Qualquer coisa

func (ph *PaymentHandler) createPayment(r *http.Request, w http.ResponseWriter) {
	data := &PaymentRequest{}
	if bindError := render.Bind(r, data); bindError != nil {
		fmt.Println(bindError.Error())
		render.Render(w, r, ErrInvalidRequest("failed to parse payment."))
		return
	}

	payment := data.Payment
	_, err := ph.db.Exec(ph.ctx, "insert into payments values ($1, $2, NOW())", payment.CorrelationId, payment.Amount)
	var pgErr *pgconn.PgError
	if err != nil {
		if errors.As(err, &pgErr) {
			fmt.Println(pgErr.Message) // => syntax error at end of input
			fmt.Println(pgErr.Code)    // => 42601
		}
		fmt.Println("err: ", err.Error())
		render.Render(w, r, ErrServerInternal())
	}

	render.Render(w, r, SuccessCreated())
}

// GET /payments/{id}

// HTTP 200 - Ok
//
//	{
//	    "correlationId": "4a7901b8-7d26-4d9d-aa19-4dc1c7cf60b3",
//	    "amount": 19.90,
//	    "requestedAt" : 2025-07-15T12:34:56.000Z
//	}
func (ph *PaymentHandler) getPayment(r *http.Request, w http.ResponseWriter) {

	if id := chi.URLParam(r, "id"); id != "" {
		pay := &PaymentResponse{Payment: &Payment{}}
		row := ph.db.QueryRow(ph.ctx, "select correlationId, amount, requested_at from payments where correlationId = $1", id).Scan(&pay.CorrelationId, &pay.Amount, &pay.RequestedAt)

		if row != nil {
			fmt.Println("err: ", row)
			render.Render(w, r, ErrServerInternal())
			return
		}
		render.Render(w, r, pay)
		return
	}

	render.Render(w, r, ErrNotFound())
}

func (ph *PaymentHandler) getPayments(r *http.Request, w http.ResponseWriter) {
	rows, err := ph.db.Query(ph.ctx, "select correlationId, amount, requested_at from payments")
	if err != nil {
		fmt.Println(err.Error())
		render.Render(w, r, ErrServerInternal())
		return
	}
	payments, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (render.Renderer, error) {
		pay := &PaymentResponse{Payment: &Payment{}}
		err := row.Scan(&pay.CorrelationId, &pay.Amount, &pay.RequestedAt)
		fmt.Println(pay.CorrelationId)
		fmt.Println(pay.Amount)
		return pay, err
	})

	if err != nil {
		fmt.Printf("CollectRows error: %v", err)
		return
	}

	if err := render.RenderList(w, r, payments); err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}
}

// GET /payments-summary?from=2020-07-10T12:34:56.000Z&to=2020-07-10T12:35:56.000Z

// HTTP 200 - Ok
// {
//     "default" : {
//         "totalRequests": 43236,
//         "totalAmount": 415542345.98
//     },
//     "fallback" : {
//         "totalRequests": 423545,
//         "totalAmount": 329347.34
//     }
// }

func (ph *PaymentHandler) getSummary(r *http.Request, w http.ResponseWriter) {}

func main() {
	flag.Parse()
	ctx := context.Background()
	dbpool, err := pgxpool.New(ctx, os.Getenv("DB_CONNECTION_STRING"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create connection pool: %v\n", err)
		os.Exit(1)
	}
	pingErr := dbpool.Ping(ctx)
	if pingErr != nil {
		fmt.Fprintf(os.Stderr, "Unable to acquire connection pool: %v\n", pingErr)
		os.Exit(1)
	}
	defer dbpool.Close()

	var greeting string

	payment := &PaymentHandler{ctx: ctx, db: dbpool}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(render.SetContentType(render.ContentTypeJSON))
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		err = dbpool.QueryRow(ctx, "select 'Hello, world!'").Scan(&greeting)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			fmt.Println(pgErr.Message) // => syntax error at end of input
			fmt.Println(pgErr.Code)    // => 42601
		}

		w.Write([]byte(greeting))
	})

	r.Get("/payments", func(w http.ResponseWriter, r *http.Request) {
		payment.getPayments(r, w)
	})

	r.Get("/payments-summary", func(w http.ResponseWriter, r *http.Request) {
		payment.getSummary(r, w)
	})

	r.Get("/payments/{id}", func(w http.ResponseWriter, r *http.Request) {
		payment.getPayment(r, w)
	})

	r.Post("/payments", func(w http.ResponseWriter, r *http.Request) {
		payment.createPayment(r, w)
	})

	if *routes {
		// fmt.Println(docgen.JSONRoutesDoc(r))
		fmt.Println(docgen.MarkdownRoutesDoc(r, docgen.MarkdownOpts{
			ProjectPath: "github.com/fabiokleis/rinha-de-backend-2025",
			Intro:       "Welcome to rinha-de-backend-2025 rest generated docs.",
		}))
		return
	}

	http.ListenAndServe(":9999", r)
}
