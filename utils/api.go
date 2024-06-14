package utils

import (
	"fmt"
	"net/http"

	"github.com/wI2L/fizz"
	"github.com/wI2L/fizz/openapi"
)

// Returns array of Fizz Operation option with summary and error responses
func CreateOperationOption(summary string, errorStatuses []int, useSecurity bool) []fizz.OperationOption {
	var option []fizz.OperationOption
	option = append(option, fizz.Summary(summary)) // append summary
	if useSecurity {
		option = append(option, fizz.Security(&openapi.SecurityRequirement{
			"bearerAuth": []string{},
		}))
	}
	for i := 0; i < len(errorStatuses); i++ { // append each error with reponse
		// create option
		option = append(option,
			fizz.Response(
				fmt.Sprint(errorStatuses[i]),
				http.StatusText(errorStatuses[i]),
				map[string]interface{}{},
				nil,
				map[string]interface{}{
					"error": "..."}))
	}
	return option
}
