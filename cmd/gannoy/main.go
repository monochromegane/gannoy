package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	flags "github.com/jessevdk/go-flags"
	"github.com/monochromegane/gannoy"
	ngt "github.com/monochromegane/go-ngt"
)

var opts Options
var createCommand CreateCommand
var dropCommand DropCommand
var saveCommand SaveCommand

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

	index, err := gannoy.NewNGTIndex(database)
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
}

func (c *DropCommand) Usage() string {
	return "[drop-OPTIONS] DATABASE"
}

type SaveCommand struct {
	Host string `short:"H" long:"host" default:"localhost" description:"Specify gannoy-db hostname."`
	Port int    `short:"P" long:"port" default:"1323" description:"Specify gannoy-db port number."`
	All  bool   `short:"A" long:"all" description:"Save all databases."`
}

func (c *SaveCommand) Execute(args []string) error {
	if !c.All && len(args) != 1 {
		return fmt.Errorf("Database name not specified.")
	}

	req, err := http.NewRequest("PUT", c.url(args), nil)
	if err != nil {
		return err
	}
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusAccepted {
		return fmt.Errorf("Saving process was not accepted.")
	}
	return nil
}

func (c *SaveCommand) url(args []string) string {
	url := fmt.Sprintf("http://%s:%d/savepoints", c.Host, c.Port)
	if !c.All {
		url += "/" + args[0]
	}
	return url
}

func (c *SaveCommand) Usage() string {
	return "[save-OPTIONS] DATABASE"
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
	parser.AddCommand("save",
		"Save database",
		"The save command register the database saving job.",
		&saveCommand)
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
