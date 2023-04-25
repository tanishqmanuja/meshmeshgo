package main

import (
	"errors"
	"os"

	"github.com/urfave/cli/v2"
)

type Config struct {
	WantHelp           bool
	SerialPortName     string
	SerialPortBaudRate int
	VerboseLevel       int
	FirmwarePath       string
	TargetNode         int
}

func NewConfig() (*Config, error) {
	var err error
	config := Config{WantHelp: true, VerboseLevel: 0}

	app := &cli.App{
		Name:  "meshmeshho",
		Usage: "meshmesh hub written in go!",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "port",
				Value:       "/dev/ttyUSB0",
				Usage:       "Serial port name",
				Destination: &config.SerialPortName,
			},
			&cli.IntFlag{
				Name:        "baud",
				Value:       460800,
				Destination: &config.SerialPortBaudRate,
			},
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
				Count:   &config.VerboseLevel,
			},
			&cli.StringFlag{
				Name:        "firmware",
				Aliases:     []string{"fw"},
				Destination: &config.FirmwarePath,
			},
			&cli.IntFlag{
				Name:        "target",
				Aliases:     []string{"t"},
				Destination: &config.TargetNode,
				Base:        16,
			},
		},
		Action: func(cCtx *cli.Context) error {
			config.WantHelp = false
			if len(config.FirmwarePath) > 0 && config.TargetNode == 0 {
				return errors.New("target node is manadatory when using firmware flag")
			}
			return nil
		},
	}

	if err = app.Run(os.Args); err != nil {
		log.Fatal(err)
	}

	return &config, err
}
