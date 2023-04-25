package main

import (
	"os"

	"github.com/urfave/cli"
)

type Config struct {
	WantHelp           bool
	SerialPortName     string
	SerialPortBaudRate int
}

func NewConfig() (*Config, error) {
	var err error
	config := Config{WantHelp: true}

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
		},
		Action: func(cCtx *cli.Context) error {
			config.WantHelp = false
			return nil
		},
	}

	if err = app.Run(os.Args); err != nil {
		log.Fatal(err)
	}

	return &config, err
}
