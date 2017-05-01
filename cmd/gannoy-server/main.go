package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/labstack/echo"
	"github.com/monochromegane/gannoy"
)

var (
	dataDir string
)

func init() {
	flag.StringVar(&dataDir, "d", ".", "Data directory.")
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
		gannoy, err := gannoy.NewGannoyIndex("hoge.meta", gannoy.Angular{}, gannoy.RandRandom{})
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

	e.Start(":1323")
}
