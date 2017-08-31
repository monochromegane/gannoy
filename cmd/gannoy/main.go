package main

import (
	"fmt"
	"os"
	"path/filepath"

	flags "github.com/jessevdk/go-flags"
	"github.com/monochromegane/gannoy"
	ngt "github.com/monochromegane/go-ngt"
)

type Options struct {
	Version bool `short:"v" long:"version" description:"Show version"`
}

type CreateCommand struct {
	Dim      int    `short:"d" long:"dim" default:"2" description:"Specify size of feature dimention."`
	Distance string `short:"D" long:"distance-function" description:"Specify distance function. [1: L1, 2: L2(default), a: angle, h: hamming]"`
	Object   string `short:"o" long:"object-type" description:"Specify object type. [f: 4 bytes float(default), c: 1 byte integer]"`
	Path     string `short:"p" long:"path" default:"." description:"Build meta file into this directory."`
}

var opts Options
var createCommand CreateCommand

func (c *CreateCommand) Execute(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("Database name not specified.")
	}

	database := filepath.Join(c.Path, args[0])
	if _, err := os.Stat(database); err == nil {
		return fmt.Errorf("Database (%s) already exists.", database)
	}

	property, err := ngt.NewNGTProperty(c.Dim)
	if err != nil {
		return err
	}
	defer property.Free()

	if c.Distance != "" {
		distanceType := ngt.DistanceTypeNone
		switch c.Distance {
		case "1":
			distanceType = ngt.DistanceTypeL1
		case "2":
			distanceType = ngt.DistanceTypeL2
		case "a":
			distanceType = ngt.DistanceTypeAngle
		case "h":
			distanceType = ngt.DistanceTypeHamming
		}
		property.SetDistanceType(distanceType)
	}

	if c.Object != "" {
		objectType := ngt.ObjectTypeNone
		switch c.Object {
		case "f":
			objectType = ngt.ObjectTypeFloat
		case "c":
			objectType = ngt.ObjectTypeUint8
		}
		property.SetObjectType(objectType)
	}

	index, err := gannoy.CreateGraphAndTree(filepath.Join(c.Path, args[0]), property)
	if err != nil {
		return err
	}
	defer index.Close()
	return index.Save()
}

func (c *CreateCommand) Usage() string {
	return "[create-OPTIONS] DATABASE"
}

func main() {
	parser := flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash) // exclude PrintError
	parser.Name = "gannoy"

	parser.AddCommand("create",
		"Create database",
		"The create command creates a meta file for the database.",
		&createCommand)
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
