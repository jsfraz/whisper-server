package routes

import (
	"fmt"
	"jsfraz/whisper-server/handlers"
	"jsfraz/whisper-server/utils"
	"net/http"

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
	engine := gin.New()
	// Limit in-memory buffering of multipart uploads; larger files stream to a temp file
	engine.MaxMultipartMemory = 8 << 20 // 8 MiB
	// Logger middleware
	if utils.GetSingleton().Config.GinMode != "release" {
		engine.Use(gin.Logger())
	} else {
		engine.Use(gin.LoggerWithConfig(gin.LoggerConfig{
			SkipPaths: []string{"/ws", "/health"},
		}))
	}
	// Recovery middleware
	engine.Use(gin.Recovery())
	// CORS config – mobile app only, no credentials needed
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
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
		infos := &openapi.Info{
			Title:       "Whisper server",
			Description: "Secure private self-hosted end-to-end encryption messaging server.",
			Version:     "1.0.0",
			Contact: &openapi.Contact{
				Name:  "Josef Ráž",
				URL:   "https://josefraz.cz",
				Email: "razj@josefraz.cz",
			},
		}
		grp.GET("openapi.json", nil, fizz.OpenAPI(infos, "json"))
	}

	// Health endpoint
	engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// WebSocket handler
	engine.GET("/ws", handlers.WebSocketHandler)

	// Setup other routes
	AuthRoute(grp)
	UserRoute(grp)
	InviteRoute(grp)
	WsAuthRoute(grp)
	FirebaseRoute(grp)
	MediaRoute(grp)

	if len(fizz.Errors()) != 0 {
		return nil, fmt.Errorf("fizz errors: %v", fizz.Errors())
	}
	return fizz, nil
}
