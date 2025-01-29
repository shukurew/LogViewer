package main

import (
	"embed"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/kevincobain2000/gol/pkg"
	"github.com/labstack/echo/v4"
)

//go:embed all:dist/*
var publicDir embed.FS

type Flags struct {
	host        string
	port        int64
	cors        int64
	every       int64
	limit       int
	baseURL     string
	filePaths   pkg.SliceFlags
	sshPaths    pkg.SliceFlags
	dockerPaths pkg.SliceFlags
	access      bool
	open        bool
	version     bool
}

var f Flags

var version = "dev"

const validToken = "AAGzTB0jI3eN26bu4OFDE99TRyjhAjBLAik" // Replace with your actual token

func main() {
	pkg.SetupLoggingStdout(slog.LevelInfo)
	flags()

	if pkg.IsInputFromPipe() {
		pkg.HandleStdinPipe()
	}
	setFilePaths()

	go pkg.WatchFilePaths(f.every, f.filePaths, f.sshPaths, f.dockerPaths, f.limit)
	slog.Info("Flags", "host", f.host, "port", f.port, "baseURL", f.baseURL, "open", f.open, "cors", f.cors, "access", f.access)

	if f.open {
		pkg.OpenBrowser(fmt.Sprintf("http://%s:%d%s", f.host, f.port, f.baseURL))
	}
	defer pkg.Cleanup()
	pkg.HandleCltrC(pkg.Cleanup)

	// Start the Echo server with token-based authentication middleware
	err := pkg.NewEcho(
		pkg.WithMiddleware(tokenAuthMiddleware), // Add your middleware here
		func(o *pkg.EchoOptions) error {
			o.Host = f.host
			o.Port = f.port
			o.Cors = f.cors
			o.Access = f.access
			o.BaseURL = f.baseURL
			o.PublicDir = &publicDir
			return nil
		},
	)
	if err != nil {
		slog.Error("starting echo", "echo", err)
		return
	}
}

// Middleware to check for valid token
func tokenAuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Skip token check for specific routes
		if c.Path() == "/favicon.ico" || strings.HasPrefix(c.Path(), "/api") {
			return next(c)
		}

		// Check token for all other routes
		token := c.QueryParam("token")
		if token != validToken {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
		}
		return next(c)
	}
}

// Rest of your code (setFilePaths, flags, wantsVersion) remains unchanged...

func setFilePaths() {
	// convenient method support for gol *logs
	if len(os.Args) > 1 {
		filePaths := pkg.SliceFlags{}
		for _, arg := range os.Args[1:] {
			// ignore background process flag
			if arg == "&" {
				continue
			}
			// Check if the argument is a flag (starts with '-')
			if strings.HasPrefix(arg, "-") {
				// If a flag is found, reset filePaths to an empty slice and break the loop
				filePaths = []string{}
				break
			}
			// Append argument to filePaths if it's not a flag
			filePaths = append(filePaths, arg)
		}
		// If filePaths is not empty, set f.filePaths to filePaths
		if len(filePaths) > 0 {
			f.filePaths = filePaths
		}
	}

	// Append GlobalPipeTmpFilePath to f.filePaths if it's not empty
	// should be set if user has piped input
	if pkg.GlobalPipeTmpFilePath != "" {
		f.filePaths = append(f.filePaths, pkg.GlobalPipeTmpFilePath)
	}

	// If f.sshPaths is not nil, process each SSH path
	if f.sshPaths != nil {
		for _, sshPath := range f.sshPaths {
			// Convert SSH path string to SSHPathConfig
			sshFilePathConfig, err := pkg.StringToSSHPathConfig(sshPath)
			if err != nil {
				slog.Error("parsing SSH path", sshPath, err)
				continue
			}
			if sshFilePathConfig != nil {
				sshConfig := pkg.SSHConfig{
					Host:           sshFilePathConfig.Host,
					Port:           sshFilePathConfig.Port,
					User:           sshFilePathConfig.User,
					Password:       sshFilePathConfig.Password,
					PrivateKeyPath: sshFilePathConfig.PrivateKeyPath,
				}
				// Get file information from the SSH path and append to GlobalFilePaths
				fileInfos := pkg.GetFileInfos(sshFilePathConfig.FilePath, f.limit, true, &sshConfig)
				pkg.GlobalFilePaths = append(pkg.GlobalFilePaths, fileInfos...)
			}
		}
	}

	// Update global file paths with the current filePaths, stdin to tmp, sshPaths, and dockerPaths
	pkg.UpdateGlobalFilePaths(f.filePaths, f.sshPaths, f.dockerPaths, f.limit)
}

func flags() {
	flag.Var(&f.filePaths, "f", "full path pattern to the log file")
	flag.Var(&f.sshPaths, "s", "full ssh path pattern to the log file")
	flag.Var(&f.dockerPaths, "d", "docker paths to the log file")
	flag.BoolVar(&f.version, "version", false, "")
	flag.BoolVar(&f.access, "access", false, "print access logs")
	flag.StringVar(&f.host, "host", "0.0.0.0", "host to serve")
	flag.Int64Var(&f.port, "port", 3003, "port to serve")
	flag.Int64Var(&f.every, "every", 10, "check for file paths every n seconds")
	flag.IntVar(&f.limit, "limit", 1000, "limit the number of files to read from the file path pattern")
	flag.Int64Var(&f.cors, "cors", 0, "cors port to allow the api (for development)")
	flag.BoolVar(&f.open, "open", true, "open browser on start")
	flag.StringVar(&f.baseURL, "base-url", "/", "base url with slash")

	flag.Parse()
	wantsVersion()
}

func wantsVersion() {
	if len(os.Args) != 2 {
		return
	}
	switch os.Args[1] {
	case "-v", "--v", "-version", "--version":
		fmt.Println(version)
		os.Exit(0)
	}
}