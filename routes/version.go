package routes

import (
	"jsfraz/whisper-server/handlers"
	"jsfraz/whisper-server/utils"
	"net/http"

	"github.com/loopfz/gadgeto/tonic"
	"github.com/wI2L/fizz"
)

// Sets the version route.
//
//	@param g
func VersionRoute(g *fizz.RouterGroup) {
	grp := g.Group("version", "Version", "Server build and client compatibility.")

	// The version is public: a client has to learn whether it is too old before
	// it can log in.
	grp.GET("",
		utils.CreateOperationOption(
			"Get server version",
			"Returns the server build, the API version and the oldest client it accepts.",
			[]int{
				http.StatusInternalServerError,
			},
			false),
		tonic.Handler(handlers.GetVersion, http.StatusOK),
	)
}
