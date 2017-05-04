package main

import (
	"context"
	"flag"
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

	"github.com/labstack/echo"
	"github.com/lestrrat/go-server-starter/listener"
	"github.com/monochromegane/gannoy"
)

var (
	dataDir           string
	withServerStarter bool
)

func init() {
	flag.StringVar(&dataDir, "d", ".", "Data directory.")
	flag.BoolVar(&withServerStarter, "s", false, "With server starter.")
	flag.Parse()
}

type Feature struct {
	W []float64 `json:"features"`
}

func main() {
	files, err := ioutil.ReadDir(dataDir)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	databases := map[string]gannoy.GannoyIndex{}
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".meta" {
			continue
		}
		key := strings.TrimSuffix(file.Name(), ".meta")
		gannoy, err := gannoy.NewGannoyIndex(filepath.Join(dataDir, file.Name()), gannoy.Angular{}, gannoy.RandRandom{})
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		databases[key] = gannoy
	}

	e := echo.New()
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
	if withServerStarter {
		address = ""
		sig = syscall.SIGTERM
		listeners, err := listener.ListenAll()
		if err != nil && err != listener.ErrNoListeningTarget {
			fmt.Println(err)
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}
}
