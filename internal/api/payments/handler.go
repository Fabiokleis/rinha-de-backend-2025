package payments

import (
	"fmt"
	"net/http"
	"time"

	cr "rinha/internal/api/common_responses"
	db "rinha/internal/database"
	prot "rinha/pkg/protocol"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/jackc/pgx/v5"
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

	payload := &prot.PaymentPayload{CorrelationId: payment.CorrelationId, Amount: payment.Amount, RequestedAt: time.Now()}
	buffer, err := payload.Encode()
	if err != nil {
		fmt.Println("failed to encode payload", err.Error())
		render.Render(w, r, cr.ErrServerInternal())
		return
	}

	_, err = db.Pgxpool.Exec(db.PgxCtx, "select pg_notify($1, $2)", prot.Payments, string(buffer))
	if err != nil {
		fmt.Println("err: ", err.Error())
		render.Render(w, r, cr.ErrServerInternal())
		return
	}

	// err := db.Pgxpool.QueryRow(db.PgxCtx, "insert into payments values ($1, $2, NOW()) returning correlation_id", payment.CorrelationId, payment.Amount).Scan(&id)
	// if err != nil {
	// 	fmt.Println("id: ", id)
	// 	fmt.Println("err: ", err.Error())
	// 	render.Render(w, r, cr.ErrServerInternal())
	// 	return
	// }

	render.Render(w, r, cr.SuccessCreated())
}

// POST /payments
// {
//     "correlationId": "4a7901b8-7d26-4d9d-aa19-4dc1c7cf60b3",
//     "amount": 19.90,
//     "requestedAt" : "2025-07-15T12:34:56.000Z"
// }

// HTTP 200 - Ok
// {
//     "message": "payment processed successfully"
// }

func (ph *PaymentHandler) processPayment(r *http.Request, w http.ResponseWriter) {
	data := &PaymentProcessRequest{}
	if bindError := render.Bind(r, data); bindError != nil {
		fmt.Println(bindError.Error())
		render.Render(w, r, cr.ErrInvalidRequest("failed to parse payment process."))
		return
	}
	payment := data.Payment
	rtx, err := db.Pgxpool.BeginTx(db.PgxCtx, pgx.TxOptions{})

	if err != nil {
		fmt.Println("failed to start transaction: ", err.Error())
		render.Render(w, r, cr.ErrServerInternal())
		return
	}

	var amount *float64
	err = rtx.QueryRow(db.PgxCtx, "select amount from payments where correlation_id = $1", payment.CorrelationId).Scan(amount)

	if err != nil {
		rtx.Rollback(db.PgxCtx)
		fmt.Println("correlation_id not found")
		render.Render(w, r, cr.ErrNotFound())
		return
	}

	var id string
	err = rtx.QueryRow(db.PgxCtx, "update table payments set amount = $2 where correlation_id = $1 returning correlation_id", payment.CorrelationId, payment.Amount-*amount).Scan(&id)

	if err != nil {
		rtx.Rollback(db.PgxCtx)
		fmt.Println("correlation_id not found")
		render.Render(w, r, cr.ErrNotFound())
		return
	}

	rtx.Commit(db.PgxCtx)
	pay := &PaymentProcessResponse{Message: "payment processed successfully"}
	render.Render(w, r, pay)
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
		row := db.Pgxpool.QueryRow(db.PgxCtx, "select correlation_id, amount, requested_at from payments where correlation_id = $1", id).Scan(&pay.CorrelationId, &pay.Amount, &pay.RequestedAt)

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
		render.Render(w, r, cr.ErrServerInternal())
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

// func (ph *PaymentHandler) getSummary(r *http.Request, w http.ResponseWriter) {

// 	from := chi.URLParam(r, "from")
// 	to := chi.URLParam(r, "to")

// 	if from != "" && to != "" {
// 		pay := &PaymentResponse{Payment: &Payment{}}
// 		row := db.Pgxpool.QueryRow(db.PgxCtx, "select correlation_id, amount, requested_at from payments where correlationId = $1", id).Scan(&pay.CorrelationId, &pay.Amount, &pay.RequestedAt)

// 		if row != nil {
// 			fmt.Println("err: ", row)
// 			render.Render(w, r, cr.ErrServerInternal())
// 			return
// 		}
// 		render.Render(w, r, pay)
// 		return
// 	}

// 	render.Render(w, r, cr.ErrNotFound())
// }
