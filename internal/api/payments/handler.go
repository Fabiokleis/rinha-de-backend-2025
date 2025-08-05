package payments

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	cr "rinha/internal/api/common_responses"
	db "rinha/internal/database"

	p "rinha/pkg/protocol"

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

	// trigger sends notification to listeners
	_, err := db.Pgxpool.Exec(db.PgxCtx, `
                    INSERT INTO payments VALUES ($1, $2, NOW(), default)`,
		payment.CorrelationId, payment.Amount)

	if err != nil {
		fmt.Println("err: ", err.Error())
		render.Render(w, r, cr.ErrServerInternal())
		return
	}

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
		pay := &PaymentResponse{Payment: &p.Payment{}}
		row := db.Pgxpool.QueryRow(db.PgxCtx, `
                         SELECT correlation_id, amount, requested_at, status
                         FROM payments
                         WHERE correlation_id = $1`,
			id).Scan(&pay.CorrelationId, &pay.Amount, &pay.RequestedAt, &pay.Status)

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
	rows, err := db.Pgxpool.Query(db.PgxCtx, `
                         SELECT correlation_id, amount, requested_at, status
                         FROM payments`)

	if err != nil {
		fmt.Println(err.Error())
		render.Render(w, r, cr.ErrServerInternal())
		return
	}
	defer rows.Close()
	payments, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (render.Renderer, error) {
		pay := &PaymentResponse{Payment: &p.Payment{}}
		err := row.Scan(&pay.CorrelationId, &pay.Amount, &pay.RequestedAt, &pay.Status)
		//fmt.Println(pay.CorrelationId)
		//fmt.Println(pay.Amount)
		return pay, err
	})

	if err != nil {
		fmt.Printf("CollectRows error: %v\n", err)
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

func (ph *PaymentHandler) getSummary(r *http.Request, w http.ResponseWriter) {

	from := chi.URLParam(r, "from")
	to := chi.URLParam(r, "to")

	var parsedFrom, parsedTo time.Time
	var err error

	if from != "" {
		parsedFrom, err = time.Parse(time.RFC3339Nano, from)
		if err != nil {
			render.Render(w, r, cr.ErrInvalidRequest("failed to parse 'from' date"))
			return
		}
	}

	if to != "" {
		parsedTo, err = time.Parse(time.RFC3339Nano, to)
		if err != nil {
			render.Render(w, r, cr.ErrInvalidRequest("failed to parse 'to' date"))
			return
		}
	}

	baseQuery := `
            SELECT
                service,
                COUNT(*) AS total_requests,
                SUM(amount) AS total_amount
            FROM
                payments
            WHERE
                service IS NOT NULL
                AND status = 'completed'`

	var conditions []string
	var args []interface{}
	argCounter := 1 // Inicia com 1 para os placeholders do pgx ($1, $2)

	if !parsedFrom.IsZero() {
		conditions = append(conditions, fmt.Sprintf("requested_at >= $%d", argCounter))
		args = append(args, from)
		argCounter++
	}

	if !parsedTo.IsZero() {
		conditions = append(conditions, fmt.Sprintf("requested_at <= $%d", argCounter))
		args = append(args, to)
		argCounter++
	}

	finalQuery := baseQuery
	if len(conditions) > 0 {
		finalQuery += " AND " + strings.Join(conditions, " AND ")
	}

	finalQuery += " GROUP BY service"

	rows, err := db.Pgxpool.Query(db.PgxCtx, finalQuery, args...)

	if err != nil {
		fmt.Println(err.Error())
		render.Render(w, r, cr.ErrServerInternal())
		return
	}

	defer rows.Close()

	summ, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (PaymentSummaryRow, error) {
		p := PaymentSummaryRow{}
		err = row.Scan(&p.Name, &p.Metric.TotalRequests, &p.Metric.TotalAmount)
		return p, err
	})

	if err != nil {
		fmt.Printf("CollectRows error: %v\n", err)
		render.Render(w, r, cr.ErrServerInternal())
		return
	}

	summary := SummaryResponse{}

	if len(summ) > 0 {
		for _, ser := range summ {
			if ser.Name == "default" {
				summary.Default = ser.Metric
			} else {
				summary.Fallback = ser.Metric
			}
		}
	} else {
		summary.Default = Service{TotalRequests: 0, TotalAmount: 0}
		summary.Fallback = summary.Default
	}

	render.Render(w, r, &summary)
	return
}

func (ph *PaymentHandler) delete(r *http.Request, w http.ResponseWriter) {

	_, err := db.Pgxpool.Exec(db.PgxCtx, "DELETE FROM processing_metrics")
	if err != nil {
		fmt.Println("err: ", err.Error())
		render.Render(w, r, cr.ErrServerInternal())
		return
	}

	_, err = db.Pgxpool.Exec(db.PgxCtx, "DELETE FROM payments")
	if err != nil {
		fmt.Println("err: ", err.Error())
		render.Render(w, r, cr.ErrServerInternal())
		return
	}

	render.Render(w, r, cr.SuccessNoContent())
}
