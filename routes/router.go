package routes

import (
	"fmt"
	"jsfraz/whisper-server/handlers"
	"jsfraz/whisper-server/utils"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/wI2L/fizz"
	"github.com/wI2L/fizz/openapi"
)

// Returns a new API router.
//
//	@return *fizz.Fizz
//	@return error
func NewRouter() (*fizz.Fizz, error) {
	// Gin instance
	engine := gin.Default()
	// Default cors config, Allow Origin, Authorization header
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowCredentials = true
	config.AllowHeaders = append(config.AllowHeaders, "Authorization")
	engine.Use(cors.New(config))

	// Fizz instance
	fizz := fizz.NewFromEngine(engine)
	// Security
	fizz.Generator().SetSecuritySchemes(map[string]*openapi.SecuritySchemeOrRef{
		"bearerAuth": {
			SecurityScheme: &openapi.SecurityScheme{
				Type:         "http",
				Scheme:       "bearer",
				BearerFormat: "JWT",
				Description:  "Put **_ONLY_** your JWT Bearer token on textbox below!",
			},
		},
	})

	// Base API route
	grp := fizz.Group("api", "", "")

	// OpenAPI spec
	if utils.GetSingleton().Config.GinMode != "release" {
		// Servers
		fizz.Generator().SetServers([]*openapi.Server{
			{
				Description: "localhost - debug",
				URL:         "http://localhost:8080",
			},
		})
		// TODO more info
		infos := &openapi.Info{
			Title:       "Whisper server",
			Description: "Secure private self-hosted end-to-end encryption messaging server.",
			Version:     "1.0.0",
			// TODO license
			Contact: &openapi.Contact{
				Name:  "Josef Ráž",
				URL:   "https://josefraz.cz",
				Email: "razj@josefraz.cz",
			},
			// TODO ToS
			// TODO XLogo
		}
		grp.GET("openapi.json", nil, fizz.OpenAPI(infos, "json"))
	}

	// TODO single use token for websocket auth
	// WebSocket handler
	engine.GET("/ws", handlers.WebSocketHandler)

	// Setup other routes
	AuthRoute(grp)
	UserRoute(grp)
	InviteRoute(grp)

	if len(fizz.Errors()) != 0 {
		return nil, fmt.Errorf("fizz errors: %v", fizz.Errors())
	}
	return fizz, nil
}
