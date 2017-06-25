package main

import (
	"fmt"
	"os"

	flags "github.com/jessevdk/go-flags"
	"github.com/monochromegane/gannoy"
)

type Options struct {
	Version bool `short:"v" long:"version" description:"Show version"`
}

type CreateCommand struct {
	Dim  int    `short:"d" long:"dim" default:"2" description:"Specify size of feature dimention."`
	Tree int    `short:"t" long:"tree" default:"1" description:"Specify size of index tree."`
	K    int    `short:"K" long:"K" default:"-1" default-mask:"twice the value of dim" description:"Specify max node size in a bucket node."`
	Path string `short:"p" long:"path" default:"." description:"Build meta file into this directory."`
}

var opts Options
var createCommand CreateCommand

func (c *CreateCommand) Execute(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("database name not specified.")
	}
	if c.K < 3 || c.K > c.Dim*2 {
		return fmt.Errorf("K must be less than dim*2 or be at least 3 or more, but %d.", c.K)
	}
	K := c.K
	if K == -1 {
		K = c.Dim * 2
	}
	err := gannoy.CreateMeta(c.Path, args[0], c.Tree, c.Dim, K)
	if err != nil {
		return err
	}
	return nil
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
