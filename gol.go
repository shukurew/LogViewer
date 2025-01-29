package gol

import (
	"embed"
	"log/slog"
	"net/http"

	"github.com/kevincobain2000/gol/pkg"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

//go:embed all:frontend/dist/*
var publicDir embed.FS

const validToken = "AAGzTB0jI3eN26bu4OFDE99TRyjhAjBLAik" // Hardcoded token

type GolOptions struct { // nolint: revive
	Every     int64
	FilePaths []string
	LogLevel  slog.Leveler
}
type GolOption func(*GolOptions) error // nolint: revive

type Gol struct {
	Options *GolOptions
}

func NewGol(opts ...GolOption) *Gol {
	options := &GolOptions{
		Every:     1000,
		LogLevel:  slog.LevelInfo,
		FilePaths: []string{},
	}
	for _, opt := range opts {
		err := opt(options)
		if err != nil {
			return nil
		}
	}
	return &Gol{
		Options: options,
	}
}

func (g *Gol) NewAPIHandler() *pkg.APIHandler {
	// Create a new Echo instance and set up token validation for /
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Token validation middleware
	e.GET("/", func(c echo.Context) error {
		token := c.QueryParam("token")
		if token != validToken {
			return echo.NewHTTPError(http.StatusUnauthorized, "Unauthorized: Invalid token")
		}
		return c.String(http.StatusOK, "Welcome to the application!")
	})

	pkg.UpdateGlobalFilePaths(g.Options.FilePaths, nil, nil, 1000)
	go pkg.WatchFilePaths(g.Options.Every, g.Options.FilePaths, nil, nil, 1000)
	return pkg.NewAPIHandler()
}

func (*Gol) NewAssetsHandler() *pkg.AssetsHandler {
	return pkg.NewAssetsHandler(&publicDir, "frontend/dist", "index.html")
}

func (*Gol) Adapter(echoHandler echo.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		e := echo.New()
		c := e.NewContext(r, w)
		if err := echoHandler(c); err != nil {
			e.HTTPErrorHandler(err, c)
		}
	}
}
