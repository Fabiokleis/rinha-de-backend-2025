package payments

import (
	"errors"
	"fmt"
	"net/http"
	cr "rinha/internal/api/common_responses"
	db "rinha/internal/database"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type PaymentHandler struct{}

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
		render.Render(w, r, cr.ErrInvalidRequest("failed to parse payment."))
		return
	}

	payment := data.Payment
	_, err := db.Pgxpool.Exec(db.PgxCtx, "insert into payments values ($1, $2, NOW())", payment.CorrelationId, payment.Amount)
	var pgErr *pgconn.PgError
	if err != nil {
		if errors.As(err, &pgErr) {
			fmt.Println(pgErr.Message) // => syntax error at end of input
			fmt.Println(pgErr.Code)    // => 42601
		}
		fmt.Println("err: ", err.Error())
		render.Render(w, r, cr.ErrServerInternal())
	}

	render.Render(w, r, cr.SuccessCreated())
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
		row := db.Pgxpool.QueryRow(db.PgxCtx, "select correlation_id, amount, requested_at from payments where correlationId = $1", id).Scan(&pay.CorrelationId, &pay.Amount, &pay.RequestedAt)

		if row != nil {
			fmt.Println("err: ", row)
			render.Render(w, r, cr.ErrServerInternal())
			return
		}
		render.Render(w, r, pay)
		return
	}

	render.Render(w, r, cr.ErrNotFound())
}

func (ph *PaymentHandler) getPayments(r *http.Request, w http.ResponseWriter) {
	rows, err := db.Pgxpool.Query(db.PgxCtx, "select correlation_id, amount, requested_at from payments")
	if err != nil {
		fmt.Println(err.Error())
		render.Render(w, r, cr.ErrServerInternal())
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
		render.Render(w, r, cr.ErrRender(err))
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
