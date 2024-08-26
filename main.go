package main

import (
	"log"
	"os"

	"github.com/terrable-dev/terrable/offline"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:    "terrable",
		Version: config()["version"],

		Commands: []*cli.Command{
			{
				Name:  "offline",
				Usage: "",
				Action: func(cCtx *cli.Context) error {
					filePath := cCtx.String("file")
					moduleName := cCtx.String("module")

					offline.Run(filePath, moduleName)
					return nil
				},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "file",
						Aliases:  []string{"f"},
						Required: true,
						Usage:    "Path to the Terraform file",
					},
					&cli.StringFlag{
						Name:     "module",
						Aliases:  []string{"m"},
						Required: true,
						Usage:    "Name of the terraform module to try and run locally",
					},
					&cli.StringFlag{
						Name:     "port",
						Aliases:  []string{"p"},
						Required: false,
						Usage:    "Name of the terraform module to try and run locally",
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
