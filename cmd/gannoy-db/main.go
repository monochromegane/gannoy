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
	"sync"
	"syscall"
	"time"

	"golang.org/x/net/netutil"

	flags "github.com/jessevdk/go-flags"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/labstack/gommon/log"
	"github.com/lestrrat/go-server-starter/listener"
	"github.com/monochromegane/conflag"
	"github.com/monochromegane/gannoy"
	"github.com/nightlyone/lockfile"
)

type Options struct {
	DataDir              string `short:"d" long:"data-dir" default:"." description:"Specify the directory where the meta files are located."`
	LogDir               string `short:"l" long:"log-dir" default-mask:"os.Stdout" description:"Specify the log output directory."`
	LockDir              string `short:"L" long:"lock-dir" default:"." description:"Specify the lock file directory. This option is used only server-starter option."`
	WithServerStarter    bool   `short:"s" long:"server-starter" description:"Use server-starter listener for server address."`
	ShutDownTimeout      int    `short:"T" long:"shutdown-timeout" default:"60" description:"Specify the number of seconds for shutdown timeout."`
	MaxConnections       int    `short:"m" long:"max-connections" default:"200" description:"Specify the number of max connections."`
	AutoSave             bool   `short:"S" long:"auto-save" description:"Automatically save the database when stopped."`
	ConcurrentToAutoSave int    `short:"C" long:"concurrent-to-auto-save" default:"5" description:"Concurrent number to auto save."`
	Thread               int    `short:"p" long:"thread" default-mask:"runtime.NumCPU()" description:"Specify number of thread."`
	Timeout              int    `short:"t" long:"timeout" default:"30" description:"Specify the number of seconds for timeout."`
	Config               string `short:"c" long:"config" default:"" description:"Configuration file path."`
	Version              bool   `short:"v" long:"version" description:"Show version"`
}

var opts Options

type Feature struct {
	W []float64 `json:"features"`
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

	// Load databases
	dirs, err := ioutil.ReadDir(opts.DataDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	thread := opts.Thread
	if thread == 0 {
		thread = runtime.NumCPU()
	}
	databases := map[string]gannoy.NGTIndex{}
	for _, dir := range dirs {
		if isDatabaseDir(dir) {
			key := dir.Name()
			index, err := gannoy.NewNGTIndex(filepath.Join(opts.DataDir, key),
				thread,
				time.Duration(opts.Timeout)*time.Second)
			if err != nil {
				e.Logger.Warnf("Database (%s) loading failed. %s", key, err)
				continue
			}
			e.Logger.Infof("Database (%s) was successfully loaded", key)
			databases[key] = index
		}
	}

	// Define API
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

		db := databases[database]
		r, err := db.SearchItem(uint(key), limit, 0.1)
		switch searchErr := err.(type) {
		case gannoy.NGTSearchError, gannoy.TimeoutError:
			e.Logger.Warnf("Search error (database: %s, key: %d): %s", database, key, searchErr)
		}
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
		bin, err := ioutil.ReadAll(c.Request().Body)
		if err != nil {
			return c.NoContent(http.StatusUnprocessableEntity)
		}

		db := databases[database]
		err = db.UpdateBinLog(key, gannoy.UPDATE, bin)
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
		db := databases[database]
		err = db.UpdateBinLog(key, gannoy.DELETE, []byte{})
		if err != nil {
			return c.NoContent(http.StatusUnprocessableEntity)
		}

		return c.NoContent(http.StatusOK)
	})

	e.GET("/health", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	e.PUT("/savepoints/:database", func(c echo.Context) error {
		database := c.Param("database")
		if _, ok := databases[database]; !ok {
			return c.NoContent(http.StatusUnprocessableEntity)
		}
		gannoy := databases[database]
		gannoy.AsyncSave()
		return c.NoContent(http.StatusAccepted)
	})

	e.PUT("/savepoints", func(c echo.Context) error {
		for _, gannoy := range databases {
			gannoy.AsyncSave()
		}
		return c.NoContent(http.StatusAccepted)
	})

	e.GET("/databases", func(c echo.Context) error {
		json := make([]string, len(databases))
		i := 0
		for key, _ := range databases {
			json[i] = key
			i += 1
		}
		return c.JSON(http.StatusOK, json)
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
			e.Logger.Info("Shutting down the server")
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, sig)
	<-sigCh

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(opts.ShutDownTimeout)*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Warn(err)
	}

	// Save databases
	if opts.AutoSave {
		save(databases, e.Logger)
	}

	// Close databases
	for _, db := range databases {
		db.Close()
	}
}

func save(databases map[string]gannoy.NGTIndex, logger echo.Logger) {
	var wg sync.WaitGroup
	wg.Add(len(databases))

	saveChan := make(chan string, len(databases))
	worker := func() {
		for key := range saveChan {
			func(wg *sync.WaitGroup) {
				defer wg.Done()
				if database, ok := databases[key]; ok {
					err := database.Save()
					if err != nil {
						logger.Errorf("Database (%s) save failed. %s", database, err)
					} else {
						logger.Infof("Database (%s) was successfully saved", database)
					}
				}
			}(&wg)
		}
	}
	for i := 0; i < opts.ConcurrentToAutoSave; i++ {
		go worker()
	}
	for key, _ := range databases {
		saveChan <- key
	}
	wg.Wait()
	close(saveChan)
}

func isDatabaseDir(dir os.FileInfo) bool {
	dbFiles := []string{"grp", "obj", "prf", "tre"}
	if !dir.IsDir() {
		return false
	}
	files, err := ioutil.ReadDir(filepath.Join(opts.DataDir, dir.Name()))
	if err != nil {
		return false
	}
	if len(files) != 4 {
		return false
	}
	for _, file := range files {
		if !contain(dbFiles, file.Name()) {
			return false
		}
	}
	return true
}

func contain(files []string, file string) bool {
	for _, f := range files {
		if file == f {
			return true
		}
	}
	return false
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

func initializeLog(logDir string) (*os.File, error) {
	if logDir == "" {
		return os.Stdout, nil
	}
	if err := os.MkdirAll(logDir, os.ModePerm); err != nil {
		return nil, err
	}
	return os.OpenFile(filepath.Join(logDir, "gannoy-db.log"), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
}
