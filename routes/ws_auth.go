package routes

import (
	"jsfraz/whisper-server/handlers"
	"jsfraz/whisper-server/middlewares"
	"jsfraz/whisper-server/utils"
	"net/http"

	"github.com/loopfz/gadgeto/tonic"
	"github.com/wI2L/fizz"
)

// Sets WS auth route group.
//
//	@param grp
func WsAuthRoute(g *fizz.RouterGroup) {

	grp := g.Group("wsauth", "WebSocket authentication", "WebSocket authentication.")
	grp.Use(middlewares.AuthMiddleware)

	// WebSocket auth
	grp.POST("",
		utils.CreateOperationOption("WebSocket auth",
			"",
			[]int{
				http.StatusBadRequest,
				http.StatusUnauthorized,
				http.StatusInternalServerError,
			},
			false),
		tonic.Handler(handlers.WebSocketAuth, http.StatusOK))
}
