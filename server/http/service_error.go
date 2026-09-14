package httpapi

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/teexue/nexa/core/service"
)

// respondServiceError maps ArgError / ServerError with errors.As, then fallback.
func respondServiceError(c *gin.Context, err error, fallback errorDetails) {
	var arg *service.ArgError
	if errors.As(err, &arg) {
		respondErrorDetails(c, errorDetails{
			Status:  http.StatusBadRequest,
			Code:    "invalid_request",
			MsgKey:  "api.error.invalid_request",
			Details: err.Error(),
		})
		return
	}
	var sev *service.ServerError
	if errors.As(err, &sev) {
		respondErrorDetails(c, serverErrorDetails(fallback, err))
		return
	}
	if fallback.Details == "" {
		fallback.Details = err.Error()
	}
	respondErrorDetails(c, fallback)
}

func respondAuthConfigError(c *gin.Context, err error) bool {
	if errors.Is(err, service.ErrStateNotConfigured) {
		respondError(c, http.StatusServiceUnavailable, "not_configured", "api.error.auth_keys_not_configured")
		return true
	}
	return false
}

func serverErrorDetails(fallback errorDetails, err error) errorDetails {
	d := fallback
	if d.Status == 0 || d.Status == http.StatusBadRequest {
		d.Status = http.StatusInternalServerError
	}
	if d.Code == "" || d.Code == "run_error" || d.Code == "optimize_error" {
		d.Code = "provider_error"
		d.MsgKey = "api.error.provider_error"
	}
	d.Details = err.Error()
	return d
}
