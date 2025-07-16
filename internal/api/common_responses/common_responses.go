package api

import (
	"net/http"

	"github.com/go-chi/render"
)

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
