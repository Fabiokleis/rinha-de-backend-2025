package payments

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/docgen"
)

func NewRouter(r *chi.Mux, gendoc bool) *PaymentHandler {
	handler := &PaymentHandler{}

	// list all payments
	r.Get("/payments", func(w http.ResponseWriter, r *http.Request) {
		handler.getPayments(r, w)
	})

	r.Get("/payments-summary", func(w http.ResponseWriter, r *http.Request) {
		handler.getSummary(r, w)
	})

	// debug payment details
	r.Get("/payments/{id}", func(w http.ResponseWriter, r *http.Request) {
		handler.getPayment(r, w)
	})

	r.Post("/payments", func(w http.ResponseWriter, r *http.Request) {
		handler.createPayment(r, w)
	})

	// implementado pelo payment processor, não pelo backend
	r.Post("/process-payment", func(w http.ResponseWriter, r *http.Request) {
		handler.processPayment(r, w)
	})

	r.Delete("/delete", func(w http.ResponseWriter, r *http.Request) {
		handler.delete(r, w)
	})

	if gendoc {
		fmt.Println(docgen.JSONRoutesDoc(r))
		// fmt.Println(docgen.MarkdownRoutesDoc(r, docgen.MarkdownOpts{
		// 	ProjectPath: "github.com/fabiokleis/rinha-de-backend-2025",
		// 	Intro:       "rinha-de-backend-2025 payments generated docs.",
		// }))
	}

	return handler
}
