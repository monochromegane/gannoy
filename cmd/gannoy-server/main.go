package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	flags "github.com/jessevdk/go-flags"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/lestrrat/go-server-starter/listener"
	"github.com/monochromegane/gannoy"
)

type Options struct {
	DataDir           string `short:"d" long:"data-dir" default:"." description:"Specify the directory where the meta files are located."`
	LogDir            string `short:"l" long:"log-dir" default-mask:"os.Stdout" description:"Specify the log output directory."`
	WithServerStarter bool   `short:"s" long:"server-starter" default:"false" description:"Use server-starter listener for server address."`
	ShutDownTimeout   int    `short:"t" long:"timeout" default:"10" description:"Specify the number of seconds for shutdown timeout."`
}

var opts Options

type Feature struct {
	W []float64 `json:"features"`
}

func main() {
	_, err := flags.Parse(&opts)
	if err != nil {
		os.Exit(1)
	}

	e := echo.New()

	log, err := initializeLog(opts.LogDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	e.Logger.SetOutput(log)
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{Output: log}))

	files, err := ioutil.ReadDir(opts.DataDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	databases := map[string]gannoy.GannoyIndex{}
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".meta" {
			continue
		}
		key := strings.TrimSuffix(file.Name(), ".meta")
		gannoy, err := gannoy.NewGannoyIndex(filepath.Join(opts.DataDir, file.Name()), gannoy.Angular{}, gannoy.RandRandom{})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		databases[key] = gannoy
	}

	e.GET("/search", func(c echo.Context) error {
		database := c.QueryParam("database")
		if _, ok := databases[database]; !ok {
			return c.NoContent(http.StatusNotFound)
		}
		key, err := strconv.Atoi(c.QueryParam("key"))
		if err != nil {
			key = -1
		}
		limit, err := strconv.Atoi(c.QueryParam("limit"))
		if err != nil {
			limit = 10
		}

		gannoy := databases[database]
		r, err := gannoy.GetNnsByKey(key, limit, -1)
		if err != nil || len(r) == 0 {
			return c.NoContent(http.StatusNotFound)
		}

		return c.JSON(http.StatusOK, r)
	})

	e.PUT("/databases/:database/features/:key", func(c echo.Context) error {
		database := c.Param("database")
		if _, ok := databases[database]; !ok {
			return c.NoContent(http.StatusUnprocessableEntity)
		}
		key, err := strconv.Atoi(c.Param("key"))
		if err != nil {
			return c.NoContent(http.StatusUnprocessableEntity)
		}
		feature := new(Feature)
		if err := c.Bind(feature); err != nil {
			return err
		}

		gannoy := databases[database]
		err = gannoy.AddItem(key, feature.W)
		if err != nil {
			return c.NoContent(http.StatusUnprocessableEntity)
		}
		return c.NoContent(http.StatusOK)
	})

	e.DELETE("/databases/:database/features/:key", func(c echo.Context) error {
		database := c.Param("database")
		if _, ok := databases[database]; !ok {
			return c.NoContent(http.StatusUnprocessableEntity)
		}
		key, err := strconv.Atoi(c.Param("key"))
		if err != nil {
			return c.NoContent(http.StatusUnprocessableEntity)
		}
		gannoy := databases[database]
		err = gannoy.RemoveItem(key)
		if err != nil {
			return c.NoContent(http.StatusUnprocessableEntity)
		}

		return c.NoContent(http.StatusOK)
	})

	address := ":1323"
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
	return os.OpenFile("access.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
}
