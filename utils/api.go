package utils

import (
	"fmt"
	"jsfraz/whisper-server/errors"

	"github.com/wI2L/fizz"
	"github.com/wI2L/fizz/openapi"
)

// Returns array of Fizz Operation option with summary and error responses
func CreateOperationOption(summary string, statuses []errors.Status, useSecurity bool) []fizz.OperationOption {
	var option []fizz.OperationOption
	option = append(option, fizz.Summary(summary)) // append summary
	if useSecurity {
		option = append(option, fizz.Security(&openapi.SecurityRequirement{
			"bearerAuth": []string{},
		}))
	}
	for i := 0; i < len(statuses); i++ { // append each error with reponse
		// error message
		message := ""
		switch statuses[i] {
		case errors.BadRequest, errors.Unauthorized, errors.InternalServerError:
			message = "..."
		default:
			message = statuses[i].GetMessage()
		}
		// create option
		option = append(option,
			fizz.Response(
				fmt.Sprint(statuses[i].GetCode()),
				statuses[i].GetMessage(),
				map[string]interface{}{},
				nil,
				map[string]interface{}{
					"error": message}))
	}
	return option
}
