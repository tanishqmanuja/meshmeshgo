package config

import (
	"os"

	"gopkg.in/ini.v1"

	"github.com/urfave/cli/v2"
	l "leguru.net/m/v2/logger"
)

type Config struct {
	WantHelp           bool
	SerialPortName     string
	SerialPortBaudRate int
	VerboseLevel       int
	TargetNode         int
	DebugNodeAddr      string
	BindAddress        string
	BindPort           int
	BasePortOffset     int
	SizeOfPortsPool    int
}

var iniConfig *ini.File

func InitINIConfig() {
	var err error
	iniConfig, err = ini.Load("meshmeshgo.ini")
	if err != nil {
		iniConfig = ini.Empty()
	}
}

func GetINIValue(section string, key string) string {
	return iniConfig.Section(section).Key(key).String()
}

func SetINIValue(section string, key string, value string) {
	iniConfig.Section(section).Key(key).SetValue(value)
	iniConfig.SaveTo("meshmeshgo.ini")
}

func NewConfig() (*Config, error) {
	var err error
	config := Config{WantHelp: true, VerboseLevel: 0, BindAddress: "dynamic", BindPort: 6053, BasePortOffset: 20000, SizeOfPortsPool: 10000}

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
			&cli.StringFlag{
				Name:        "node_to_debug",
				Aliases:     []string{"dbg"},
				Usage:       "Debug a single node connection",
				Destination: &config.DebugNodeAddr,
			},
			&cli.StringFlag{
				Name:        "bind_address",
				Value:       "dynamic",
				Usage:       "Bind address for the esphome servers. Use 'dynamic' to auto-assign a port based on the node id",
				Destination: &config.BindAddress,
			},
			&cli.IntFlag{
				Name:        "bind_port",
				Value:       6053,
				Usage:       "Bind port for the esphome servers. Use 0 to auto-assign a port based on the bind address",
				Destination: &config.BindPort,
			},
			&cli.IntFlag{
				Name:        "base_port_offset",
				Value:       20000,
				Usage:       "Base port offset for the esphome servers",
				Destination: &config.BasePortOffset,
			},
			&cli.IntFlag{
				Name:        "size_of_ports_pool",
				Value:       10000,
				Usage:       "Size of ports pool for the server",
				Destination: &config.SizeOfPortsPool,
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
