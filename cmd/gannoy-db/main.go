package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/net/netutil"

	"github.com/jessevdk/go-flags"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/labstack/gommon/log"
	"github.com/lestrrat/go-server-starter/listener"
	"github.com/monochromegane/conflag"
	"github.com/monochromegane/gannoy"
	"github.com/nightlyone/lockfile"
)

type Options struct {
	DataDir           string `short:"d" long:"data-dir" default:"." description:"Specify the directory where the meta files are located."`
	LogDir            string `short:"l" long:"log-dir" default-mask:"os.Stdout" description:"Specify the log output directory."`
	LockDir           string `short:"L" long:"lock-dir" default:"." description:"Specify the lock file directory. This option is used only server-starter option."`
	WithServerStarter bool   `short:"s" long:"server-starter" description:"Use server-starter listener for server address."`
	ShutDownTimeout   int    `short:"t" long:"timeout" default:"10" description:"Specify the number of seconds for shutdown timeout."`
	MaxConnections    int    `short:"m" long:"max-connections" default:"100" description:"Specify the number of max connections."`
	Config            string `short:"c" long:"config" default:"" description:"Configuration file path."`
	Version           bool   `short:"v" long:"version" description:"Show version"`
}

var opts Options

type Feature struct {
	W []float64 `json:"features"`
}

type FeatureWithKey struct {
	Key int       `json:"key"`
	W   []float64 `json:"features"`
}

func main() {

	// Parse option from args and configuration file.
	conflag.LongHyphen = true
	conflag.BoolValue = false
	parser := flags.NewParser(&opts, flags.Default)
	_, err := parser.ParseArgs(os.Args[1:])
	if err != nil {
		os.Exit(1)
	}
	if opts.Version {
		fmt.Printf("%s version %s\n", parser.Name, gannoy.VERSION)
		os.Exit(0)
	}
	if opts.Config != "" {
		if args, err := conflag.ArgsFrom(opts.Config); err == nil {
			if _, err := parser.ParseArgs(args); err != nil {
				os.Exit(1)
			}
		}
	}
	_, err = parser.ParseArgs(os.Args[1:])
	if err != nil {
		os.Exit(1)
	}

	// Wait old process finishing.
	if opts.WithServerStarter {
		lock, err := initializeLock(opts.LockDir)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		defer lock.Unlock()
		for {
			if err := lock.TryLock(); err != nil {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			break
		}
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

	// Load meta files
	files, err := ioutil.ReadDir(opts.DataDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	metaCh := make(chan string, len(files))
	gannoyCh := make(chan gannoy.GannoyIndex)
	errCh := make(chan error)
	databases := map[string]gannoy.GannoyIndex{}
	var metaCount int
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".meta" {
			continue
		}
		metaCh <- filepath.Join(opts.DataDir, file.Name())
		metaCount++
	}
	if metaCount == 0 {
		fmt.Fprintln(os.Stderr, "Do not exist Meta files.")
		close(metaCh)
		close(gannoyCh)
		close(errCh)
		os.Exit(1)
	}

	for i := 0; i < runtime.GOMAXPROCS(0); i++ {
		go gannoyIndexInitializer(metaCh, gannoyCh, errCh)
	}

loop:
	for {
		select {
		case gannoy := <-gannoyCh:
			key := strings.TrimSuffix(filepath.Base(gannoy.MetaFile()), ".meta")
			databases[key] = gannoy
			if len(databases) >= metaCount {
				close(metaCh)
				close(gannoyCh)
				close(errCh)
				break loop
			}
		case err := <-errCh:
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	// Define API
	e.GET("/search/:database/:key", func(c echo.Context) error {
		database := c.Param("database")
		key, err := strconv.Atoi(c.Param("key"))
		if err != nil {
			key = -1
		}
		limit, err := strconv.Atoi(c.QueryParam("limit"))
		if err != nil {
			limit = 10
		}

		if gannoy, ok := databases[database]; ok {
			if r, err := gannoy.GetNnsByKey(key, limit, -1); err == nil {
				return c.JSON(http.StatusOK, r)
			} else {
				return c.JSON(http.StatusOK, err)
			}
		}

		return c.NoContent(http.StatusNotFound)
	})

	e.POST("/search/:database", func(c echo.Context) error {
		database := c.Param("database")
		feature := new(Feature)
		if err := c.Bind(feature); err != nil {
			return err
		}
		limit, err := strconv.Atoi(c.QueryParam("limit"))
		if err != nil {
			limit = 10
		}
		if gannoy, ok := databases[database]; ok {
			if r, err := gannoy.GetAllNns(feature.W, limit, -1); err == nil {
				return c.JSON(http.StatusOK, r)
			} else {
				return c.JSON(http.StatusOK, err)
			}
		}
		return c.NoContent(http.StatusNotFound)
	})

	e.POST("/databases/:database/features", func(c echo.Context) error {
		database := c.Param("database")
		if _, ok := databases[database]; !ok {
			return c.NoContent(http.StatusUnprocessableEntity)
		}
		feature := new(FeatureWithKey)
		if err := c.Bind(feature); err != nil {
			return err
		}

		gannoy := databases[database]
		err = gannoy.AddItem(feature.Key, feature.W)
		if err != nil {
			return c.NoContent(http.StatusUnprocessableEntity)
		}
		return c.NoContent(http.StatusOK)
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
		err = gannoy.UpdateItem(key, feature.W)
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

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, databases)
		//return c.NoContent(http.StatusOK)
	})

	// Start server
	sig := os.Interrupt
	if opts.WithServerStarter {
		sig = syscall.SIGTERM
		listeners, err := listener.ListenAll()
		if err != nil && err != listener.ErrNoListeningTarget {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		e.Listener = netutil.LimitListener(listeners[0], opts.MaxConnections)
	} else {
		l, err := net.Listen("tcp", ":1323")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		e.Listener = netutil.LimitListener(l, opts.MaxConnections)
	}

	go func() {
		if err := e.Start(""); err != nil {
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
	return os.OpenFile(filepath.Join(logDir, "gannoy-db.log"), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
}

func initializeLock(lockDir string) (lockfile.Lockfile, error) {
	if err := os.MkdirAll(lockDir, os.ModePerm); err != nil {
		return "", err
	}
	lock := "gannoy-db.lock"
	if !filepath.IsAbs(lockDir) {
		lockDir, err := filepath.Abs(lockDir)
		if err != nil {
			return lockfile.Lockfile(""), err
		}
		return lockfile.New(filepath.Join(lockDir, lock))
	}
	return lockfile.New(filepath.Join(lockDir, lock))
}

func gannoyIndexInitializer(metaCh chan string, gannoyCh chan gannoy.GannoyIndex, errCh chan error) {
	for meta := range metaCh {
		gannoy, err := gannoy.NewGannoyIndex(meta, gannoy.Angular{}, gannoy.RandRandom{})
		if err == nil {
			gannoyCh <- gannoy
		} else {
			errCh <- err
		}
	}
}
