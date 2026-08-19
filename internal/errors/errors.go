package errors

import (
	"errors"
	"fmt"
)

type APIError struct {
	Status  int
	Body    any
	Message string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("KlickTipp API request failed with status %d.", e.Status)
}

func Structured(err error) map[string]any {
	if err == nil {
		return nil
	}

	var apiErr *APIError
	if errors.As(err, &apiErr) {
		errType := "klicktipp_api_error"
		if apiErr.Status == 406 {
			errType = "business_validation_error"
		}

		return map[string]any{
			"type":    errType,
			"status":  apiErr.Status,
			"message": apiErr.Error(),
			"body":    apiErr.Body,
		}
	}

	return map[string]any{
		"type":    "internal_error",
		"message": err.Error(),
	}
}
