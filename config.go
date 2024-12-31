package main

import (
	"os"

	"github.com/urfave/cli/v2"
	l "leguru.net/m/v2/logger"
)

type Config struct {
	WantHelp           bool
	SerialPortName     string
	SerialPortBaudRate int
	VerboseLevel       int
	TargetNode         int
	Discovery          bool
	DebugNodeAddr      string
}

func NewConfig() (*Config, error) {
	var err error
	config := Config{WantHelp: true, VerboseLevel: 0}

	app := &cli.App{
		Name:  "meshmeshgo",
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
			&cli.IntFlag{
				Name:        "target",
				Aliases:     []string{"t"},
				Destination: &config.TargetNode,
				Base:        16,
			},
			&cli.BoolFlag{
				Name:        "discovery",
				Value:       false,
				Usage:       "Execute a round of discovey on the network",
				Aliases:     []string{"d"},
				Destination: &config.Discovery,
			},
			&cli.StringFlag{
				Name:        "node_to_debug",
				Aliases:     []string{"dbg"},
				Usage:       "Debug a single node connection",
				Destination: &config.DebugNodeAddr,
			},
		},
		Action: func(cCtx *cli.Context) error {
			config.WantHelp = false
			return nil
		},
	}

	if err = app.Run(os.Args); err != nil {
		l.Log().Fatal(err)
	}

	return &config, err
}
