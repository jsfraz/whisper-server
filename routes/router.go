package routes

import (
	"fmt"
	"jsfraz/shept-server/handlers"
	"jsfraz/shept-server/middlewares"
	"jsfraz/shept-server/utils"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"

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
	// Only the local reverse proxy may set the client address; otherwise any
	// client could rotate X-Forwarded-For and dodge the rate limit.
	if err := engine.SetTrustedProxies(utils.GetSingleton().Config.TrustedProxyList()); err != nil {
		return nil, fmt.Errorf("trusted proxies: %w", err)
	}
	// Limit in-memory buffering of multipart uploads; larger files stream to a temp file
	engine.MaxMultipartMemory = 8 << 20 // 8 MiB
	// Logger middleware (structured, request id)
	engine.Use(middlewares.RequestLogger())
	// Recovery middleware
	engine.Use(gin.Recovery())
	// CORS config – mobile app only, no credentials needed
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowHeaders = append(config.AllowHeaders, "Authorization")
	engine.Use(cors.New(config))

	// Error bodies carry a translatable code, never a server sentence.
	tonic.SetErrorHook(utils.TonicErrorHook)

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
			Title:       "Shept server",
			Description: "Secure private self-hosted end-to-end encryption messaging server.",
			Version:     "0.5.0",
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
	VersionRoute(grp)
	AuthRoute(grp)
	UserRoute(grp)
	MessageRoute(grp)
	ChatsRoute(grp)
	InviteRoute(grp)
	WsAuthRoute(grp)
	FirebaseRoute(grp)
	MediaRoute(grp)

	// Document the media endpoints, which are raw Gin handlers (see media_openapi.go).
	addMediaOperations(fizz.Generator())

	if len(fizz.Errors()) != 0 {
		return nil, fmt.Errorf("fizz errors: %v", fizz.Errors())
	}

	return fizz, nil
}
