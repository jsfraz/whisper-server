package utils

import (
	"fmt"
	"net/http"

	"github.com/wI2L/fizz"
	"github.com/wI2L/fizz/openapi"
)

// Returns array of Fizz Operation option with summary and error responses
//
//	@param summary
//	@param description
//	@param errorStatuses
//	@param useSecurity
//	@return []fizz.OperationOption
func CreateOperationOption(summary string, description string, errorStatuses []int, useSecurity bool) []fizz.OperationOption {
	var option []fizz.OperationOption
	option = append(option, fizz.Summary(summary)) // Append summary
	// Append description
	if description != "" {
		option = append(option, fizz.Description(description))
	}
	if useSecurity {
		option = append(option, fizz.Security(&openapi.SecurityRequirement{
			"bearerAuth": []string{},
		}))
	}
	for i := 0; i < len(errorStatuses); i++ { // Append each error with reponse
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
