package main

import "github.com/urfave/cli/v2"

var (
	ConfigFlag = &cli.StringFlag{
		Name:    "config",
		Aliases: []string{"c"},
		Value:   "config.json",
		Usage:   "load configuration from `file`",
	}
	CollectionUpdateAllFlag = &cli.BoolFlag{
		Name:  "all",
		Usage: "update all collections",
	}
	CollectionUpdateNFlag = &cli.IntFlag{
		Name:  "n",
		Usage: "update top `n` collections",
	}
	CollectionUpdateDaemonFlag = &cli.BoolFlag{
		Name:  "daemon",
		Usage: "run as daemon",
	}
	CollectionUpdateFromFlag = &cli.IntFlag{
		Name:  "from",
		Usage: "start update from this offset",
	}
	CollectionUpdateToFlag = &cli.IntFlag{
		Name:  "to",
		Usage: "end update to this offset",
	}
	CollectionUpdateLimitFlag = &cli.IntFlag{
		Name:  "limit",
		Value: 300,
		Usage: "one page `limit`",
	}

	CollectionListNFlag = &cli.IntFlag{
		Name:  "n",
		Usage: "list top `N` collections",
	}
)
