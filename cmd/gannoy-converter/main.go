package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"runtime"

	flags "github.com/jessevdk/go-flags"
	"github.com/monochromegane/gannoy"
)

type Options struct {
	Dim     int    `short:"d" long:"dim" default:"2" description:"Specify size of feature dimention."`
	Path    string `short:"p" long:"path" default:"." description:"Build meta file into this directory."`
	Maps    string `short:"m" long:"map-path" default:"" description:"Specify key and index mapping CSV file, if exist."`
	Thread  int    `short:"t" long:"thread" default-mask:"runtime.NumCPU()" description:"Specify number of thread."`
	Version bool   `short:"v" long:"version" description:"Show version"`
}

var opts Options

func main() {
	parser := flags.NewParser(&opts, flags.Default)
	parser.Usage = "[OPTIONS] SRC_ANNOY_OR_CSV_FILE DEST_DATABASE_NAME"
	args, err := parser.Parse()
	if err != nil {
		os.Exit(1)
	}
	if opts.Version {
		fmt.Printf("%s version %s\n", parser.Name, gannoy.VERSION)
		os.Exit(0)
	}
	if len(args) != 2 {
		fmt.Fprintf(os.Stderr, "source annoy or CSV file and destination database name not specified.\n")
		os.Exit(1)
	}

	thread := opts.Thread
	if thread == 0 {
		thread = runtime.NumCPU()
	}
	converter := gannoy.NewConverter(args[0], opts.Dim, thread, binary.LittleEndian)
	err = converter.Convert(args[0], opts.Path, args[1], opts.Maps)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
