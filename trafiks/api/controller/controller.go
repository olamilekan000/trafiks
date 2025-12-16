package controller

import (
	"strings"

	"go.uber.org/fx"

	"github.com/go-playground/validator/v10"
)

var Module = fx.Options(
	fx.Provide(NewAuth),
	fx.Provide(NewUser),
	fx.Provide(NewAPIKey),
	fx.Provide(NewProject),
	fx.Provide(NewService),
	fx.Provide(NewProxyRequestLog),
	fx.Provide(NewMetrics),
	fx.Provide(NewWebhook),
)

func translateValidationErrors(err error) string {
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		msg := make([]string, 0)
		for _, fieldErr := range validationErrors {
			switch fieldErr.Field() {
			case "OldPassword":
				msg = append(msg, "Old password is required.")
			case "NewPassword":
				switch fieldErr.Tag() {
				case "required":
					msg = append(msg, "New password is required.")
				case "min":
					msg = append(msg, "New password must be at least 8 characters.")
				}
			}
		}

		return strings.Join(msg, " ")
	}

	return "invalid request"
}
