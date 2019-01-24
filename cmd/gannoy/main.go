package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	flags "github.com/jessevdk/go-flags"
	"github.com/monochromegane/gannoy"

	ngt "github.com/yahoojapan/gongt"
)

var opts Options
var createCommand CreateCommand
var dropCommand DropCommand
var applyCommand ApplyCommand

type Options struct {
	Version bool `short:"v" long:"version" description:"Show version"`
}

type CreateCommand struct {
	Dim      int    `short:"d" long:"dim" default:"2" description:"Specify size of feature dimention."`
	Distance string `short:"D" long:"distance-function" description:"Specify distance function. [1: L1, 2: L2(default), a: angle, h: hamming]"`
	Object   string `short:"o" long:"object-type" description:"Specify object type. [f: 4 bytes float(default), c: 1 byte integer]"`
	Path     string `short:"p" long:"path" default:"." description:"Build meta file into this directory."`
}

func (c *CreateCommand) Execute(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("Database name not specified.")
	}

	database := filepath.Join(c.Path, args[0])
	if _, err := os.Stat(database); err == nil {
		return fmt.Errorf("Database (%s) already exists.", database)
	}

	idx := ngt.New(database)

	idx.SetDimension(c.Dim)
	if c.Distance != "" {
		distanceType := ngt.DistanceNone
		switch c.Distance {
		case "1":
			distanceType = ngt.L1
		case "2":
			distanceType = ngt.L2
		case "a":
			distanceType = ngt.Angle
		case "h":
			distanceType = ngt.Hamming
		}
		idx.SetDistanceType(distanceType)
	}

	if c.Object != "" {
		objectType := ngt.ObjectNone
		switch c.Object {
		case "f":
			objectType = ngt.Float
		case "c":
			objectType = ngt.Uint8
		}
		idx.SetObjectType(objectType)
	}

	index, err := gannoy.CreateGraphAndTree(filepath.Join(c.Path, args[0]), idx)
	if err != nil {
		return err
	}
	defer index.Close()
	return nil
}

func (c *CreateCommand) Usage() string {
	return "[create-OPTIONS] DATABASE"
}

type DropCommand struct {
	Yes  bool   `short:"y" long:"assumeyes" description:"Answer yes for all questions."`
	Path string `short:"p" long:"path" default:"." description:"Build meta file into this directory."`
}

func (c *DropCommand) Execute(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("Database name not specified.")
	}

	database := filepath.Join(c.Path, args[0])
	if _, err := os.Stat(database); err != nil {
		return fmt.Errorf("Database (%s) dose not exist.", database)
	}

	index, err := gannoy.NewNGTIndexMeta(database, 1, 1)
	if err != nil {
		return err
	}

	if c.Yes {
		return index.Drop()
	} else {
		var confirm string
		fmt.Printf("Do you want to drop the database? (%s) [y|n]\n", database)
		fmt.Scan(&confirm)
		if confirm == "y" || confirm == "yes" {
			return index.Drop()
		}
		index.Close()
		return nil
	}
	return nil
}

func (c *DropCommand) Usage() string {
	return "[drop-OPTIONS] DATABASE"
}

type ApplyCommand struct {
	Path string `short:"p" long:"path" default:"." description:"Build meta file into this directory."`
}

func (c *ApplyCommand) Usage() string {
	return "[apply-OPTIONS] DATABASE"
}

func (c *ApplyCommand) Execute(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("Database name not specified.")
	}

	database := filepath.Join(c.Path, args[0])
	if _, err := os.Stat(database); err != nil {
		return fmt.Errorf("Database (%s) dose not exist.", database)
	}

	index, err := gannoy.NewNGTIndexMeta(database, runtime.NumCPU(), 0)
	if err != nil {
		return err
	}
	return index.Apply()
}

func main() {
	parser := flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash) // exclude PrintError
	parser.Name = "gannoy"

	parser.AddCommand("create",
		"Create database",
		"The create command creates a meta file for the database.",
		&createCommand)
	parser.AddCommand("drop",
		"Drop database",
		"The drop command drops the database.",
		&dropCommand)
	parser.AddCommand("apply",
		"Apply database",
		"The apply command update the database.",
		&applyCommand)
	_, err := parser.Parse()
	if err != nil {
		if opts.Version && err.(*flags.Error).Type == flags.ErrCommandRequired {
			fmt.Printf("%s version %s\n", parser.Name, gannoy.VERSION)
			os.Exit(0)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
