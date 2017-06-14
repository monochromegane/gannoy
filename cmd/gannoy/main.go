package main

import (
	"fmt"
	"os"

	flags "github.com/jessevdk/go-flags"
	"github.com/monochromegane/gannoy"
)

type CreateCommand struct {
	Dim  int    `short:"d" long:"dim" default:"2" description:"Specify size of feature dimention."`
	Tree int    `short:"t" long:"tree" default:"1" description:"Specify size of index tree."`
	Path string `short:"p" long:"path" default:"." description:"Build meta file into this directory."`
}

var createCommand CreateCommand

func (c *CreateCommand) Execute(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("database name not specified.")
	}
	err := gannoy.CreateMeta(c.Path, args[0], c.Tree, c.Dim, c.Dim*2)
	if err != nil {
		return err
	}
	return nil
}
func (c *CreateCommand) Usage() string {
	return "[create-OPTIONS] DATABASE"
}

func main() {
	parser := flags.NewParser(nil, flags.Default)
	parser.Name = "gannoy"

	parser.AddCommand("create",
		"Create database",
		"The create command creates a meta file for the database.",
		&createCommand)
	_, err := parser.Parse()
	if err != nil {
		os.Exit(1)
	}
}
