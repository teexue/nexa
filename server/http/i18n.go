package httpapi

import (
	"github.com/gin-gonic/gin"

	"github.com/teexue/nexa/core/i18n"
)

// respondError writes a localized JSON error body.
func respondError(c *gin.Context, status int, code, msgKey string, args ...any) {
	msg := i18n.TCtx(c.Request.Context(), msgKey, args...)
	c.JSON(status, gin.H{"code": code, "message": msg})
}

type errorDetails struct {
	Status  int
	Code    string
	MsgKey  string
	Details string
	Args    []any
}

func respondErrorDetails(c *gin.Context, d errorDetails) {
	msg := i18n.TCtx(c.Request.Context(), d.MsgKey, d.Args...)
	body := gin.H{"code": d.Code, "message": msg}
	if d.Details != "" {
		body["details"] = d.Details
	}
	c.JSON(d.Status, body)
}

// LocaleMiddleware parses Accept-Language and attaches an i18n bundle to the request context.
func LocaleMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		locale := i18n.ParseAcceptLanguage(c.GetHeader("Accept-Language"))
		if locale == "" {
			locale = i18n.Global().Locale()
		}
		bundle, err := i18n.NewBundle(locale)
		if err != nil {
			bundle = i18n.Global()
		}
		c.Request = c.Request.WithContext(i18n.WithLocale(c.Request.Context(), bundle))
		c.Next()
	}
}
