package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/monochromegane/gannoy"
)

var (
	path     string
	database string
	dim      int
	tree     int
	K        int
)

func init() {
	flag.StringVar(&path, "p", ".", "Database path.")
	flag.StringVar(&database, "d", "", "Database name.")
	flag.IntVar(&dim, "dim", 2, "Dimention.")
	flag.IntVar(&tree, "tree", 1, "Tree.")
	flag.IntVar(&K, "K", 50, "K.")
	flag.Parse()
}

func main() {
	err := gannoy.CreateMeta(path, database, tree, dim, K)
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
}
