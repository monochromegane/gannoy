package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	flags "github.com/jessevdk/go-flags"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/labstack/gommon/log"
	"github.com/lestrrat/go-server-starter/listener"
)

type Options struct {
	Db                string `long:"db" default:":1323" description:"Specify gannoy-db address."`
	LogDir            string `short:"l" long:"log-dir" default-mask:"os.Stdout" description:"Specify the log output directory."`
	WithServerStarter bool   `short:"s" long:"server-starter" default:"false" description:"Use server-starter listener for server address."`
	ShutDownTimeout   int    `short:"t" long:"timeout" default:"10" description:"Specify the number of seconds for shutdown timeout."`
}

var opts Options

func main() {
	_, err := flags.Parse(&opts)
	if err != nil {
		os.Exit(1)
	}

	e := echo.New()

	// initialize log
	l, err := initializeLog(opts.LogDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	e.Logger.SetLevel(log.INFO)
	e.Logger.SetOutput(l)
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{Output: l}))

	// Reverse proxy
	rp := &httputil.ReverseProxy{Director: func(request *http.Request) {
		request.URL.Scheme = "http"
		request.URL.Host = opts.Db
	},
	}
	e.GET("/search", echo.WrapHandler(rp))
	e.PUT("/databases/:database/features/:key", echo.WrapHandler(rp))
	e.DELETE("/databases/:database/features/:key", echo.WrapHandler(rp))

	// Start server
	address := ":1324"
	sig := os.Interrupt
	if opts.WithServerStarter {
		address = ""
		sig = syscall.SIGTERM
		listeners, err := listener.ListenAll()
		if err != nil && err != listener.ErrNoListeningTarget {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		e.Listener = listeners[0]
	}

	go func() {
		if err := e.Start(address); err != nil {
			e.Logger.Info("shutting down the server")
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, sig)
	<-sigCh

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(opts.ShutDownTimeout)*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}
}

func initializeLog(logDir string) (*os.File, error) {
	if logDir == "" {
		return os.Stdout, nil
	}
	if err := os.MkdirAll(logDir, os.ModePerm); err != nil {
		return nil, err
	}
	return os.OpenFile(filepath.Join(logDir, "access.log"), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
}
