package pkg

import (
	"embed"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type EchoOptions struct {
	Host      string
	Port      int64
	Cors      int64
	BaseURL   string
	Access    bool
	PublicDir *embed.FS
	Middleware []echo.MiddlewareFunc // Add this field
}

type EchoOption func(*EchoOptions) error

func NewEcho(opts ...EchoOption) error {
	options := &EchoOptions{
		Cors:      0,
		BaseURL:   "/",
		Host:      "localhost", // default host
		Port:      3000,        // default port
		Access:    false,
		PublicDir: nil,
	}
	for _, opt := range opts {
		err := opt(options)
		if err != nil {
			return err
		}
	}
	e := echo.New()

	SetupMiddlewares(e)
	if options.Access {
		e.Use(middleware.Logger())
	}

	// Apply custom middleware if provided
	if options.Middleware != nil {
		for _, m := range options.Middleware {
			e.Use(m)
		}
	}

	SetupRoutes(e, options)
	SetupCors(e, options)

	e.Logger.Fatal(e.Start(fmt.Sprintf("%s:%d", options.Host, options.Port)))
	return nil
}

func WithMiddleware(middleware ...echo.MiddlewareFunc) EchoOption {
	return func(o *EchoOptions) error {
		o.Middleware = middleware
		return nil
	}
}

func SetupMiddlewares(e *echo.Echo) {
	e.HTTPErrorHandler = HTTPErrorHandler
	e.Use(middleware.Recover())
	e.Use(middleware.Gzip())
	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: ltsv(),
	}))
}

func SetupRoutes(e *echo.Echo, options *EchoOptions) {
	e.GET(options.BaseURL+"", NewAssetsHandler(options.PublicDir, "dist", "index.html").Get)
	e.GET(options.BaseURL+"favicon.ico", NewAssetsHandler(options.PublicDir, "dist", "favicon.ico").GetICO)
	e.GET(options.BaseURL+"api", NewAPIHandler().Get)
}

func SetupCors(e *echo.Echo, options *EchoOptions) {
	if options.Cors == 0 {
		return
	}
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{fmt.Sprintf("http://localhost:%d", options.Cors)},
		AllowMethods: []string{http.MethodGet, http.MethodPut, http.MethodPost, http.MethodDelete},
	}))
}

// HTTPErrorResponse is the response for HTTP errors
type HTTPErrorResponse struct {
	Error interface{} `json:"error"`
}

// HTTPErrorHandler handles HTTP errors for entire application
func HTTPErrorHandler(err error, c echo.Context) {
	SetHeadersResponseJSON(c.Response().Header())
	code := http.StatusInternalServerError
	var message interface{}
	// nolint: errorlint
	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
		message = he.Message
	} else {
		message = err.Error()
	}

	if code == http.StatusInternalServerError {
		message = fmt.Sprintf("%v", err)
	}
	if err = c.JSON(code, &HTTPErrorResponse{Error: message}); err != nil {
		slog.Error("handling HTTP error", "handler", err)
	}
}

func ltsv() string {
	timeCustom := time.Now().Format("2006-01-02 15:04:05")
	var format string
	format += fmt.Sprintf("time:%s\t", timeCustom)
	format += "host:${remote_ip}\t"
	format += "forwardedfor:${header:x-forwarded-for}\t"
	format += "req:-\t"
	format += "status:${status}\t"
	format += "method:${method}\t"
	format += "uri:${uri}\t"
	format += "size:${bytes_out}\t"
	format += "referer:${referer}\t"
	format += "ua:${user_agent}\t"
	format += "reqtime_ns:${latency}\t"
	format += "cache:-\t"
	format += "runtime:-\t"
	format += "apptime:-\t"
	format += "vhost:${host}\t"
	format += "reqtime_human:${latency_human}\t"
	format += "x-request-id:${id}\t"
	format += "host:${host}\n"
	return format
}