package routes

import (
	"fmt"
	"jsfraz/whisper-server/handlers"
	"jsfraz/whisper-server/utils"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/wI2L/fizz"
	"github.com/wI2L/fizz/openapi"
)

const staticRoot = "static"

var staticAbsRoot string

// noDirFS wraps an http.FileSystem and rejects directory opens (no listing).
type noDirFS struct {
	fs http.FileSystem
}

func (fs noDirFS) Open(name string) (http.File, error) {
	f, err := fs.fs.Open(name)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	if info.IsDir() {
		_ = f.Close()
		return nil, os.ErrNotExist
	}
	return f, nil
}

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

	// Static files - must be after defining all API routes
	absRoot, err := filepath.Abs(staticRoot)
	if err != nil {
		return nil, fmt.Errorf("static root: %w", err)
	}
	staticAbsRoot = absRoot
	engine.StaticFS("/static", noDirFS{http.Dir(staticRoot)})
	engine.GET("/", func(c *gin.Context) {
		path, err := resolveStaticPath("index.html")
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		c.File(path)
	})
	engine.NoRoute(serveStaticFile)

	if len(fizz.Errors()) != 0 {
		return nil, fmt.Errorf("fizz errors: %v", fizz.Errors())
	}

	return fizz, nil
}

// resolveStaticPath maps a file name under static/ and ensures the result stays inside staticAbsRoot.
func resolveStaticPath(name string) (string, error) {
	clean := filepath.Clean(name)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return "", os.ErrNotExist
	}

	path := filepath.Join(staticRoot, clean)
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if abs != staticAbsRoot && !strings.HasPrefix(abs, staticAbsRoot+string(os.PathSeparator)) {
		return "", os.ErrNotExist
	}

	info, err := os.Stat(abs)
	if err != nil || info.IsDir() {
		return "", os.ErrNotExist
	}

	return abs, nil
}

// serveStaticFile serves top-level files from static/ (e.g. /icon.svg).
// Unmatched API paths are not affected — they contain a slash in the path segment.
func serveStaticFile(c *gin.Context) {
	if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
		c.Status(http.StatusNotFound)
		return
	}

	name := strings.TrimPrefix(c.Request.URL.Path, "/")
	if name == "" || strings.Contains(name, "/") || strings.Contains(name, "..") {
		c.Status(http.StatusNotFound)
		return
	}

	path, err := resolveStaticPath(name)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	c.File(path)
}
