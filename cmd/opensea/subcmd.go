package main

import (
	"github.com/urfave/cli/v2"
	"github.com/xyths/hs"
	"github.com/xyths/opensea-monitor/opensea"
)

var (
	collectionCommand = &cli.Command{
		//Action: collection,
		Name:  "collection",
		Usage: "Update collections",
		Subcommands: []*cli.Command{
			{
				Action: updateCollection,
				Name:   "update",
				Usage:  "Update collections on OpenSea",
				Flags: []cli.Flag{
					CollectionUpdateAllFlag,
					CollectionUpdateNFlag,
					CollectionUpdateDaemonFlag,
					//CollectionUpdateFromFlag,
					//CollectionUpdateToFlag,
					//CollectionUpdateLimitFlag,
				},
			},
			{
				Action: listTopCollection,
				Name:   "top",
				Usage:  "List top N collections on OpenSea",
				Flags: []cli.Flag{
					CollectionListNFlag,
				},
			},
		},
	}
	eventCommand = &cli.Command{
		Name:  "event",
		Usage: "Manage OpenSea events of the NFTs",
		Subcommands: []*cli.Command{
			{
				Action: monitor,
				Name:   "monitor",
				Usage:  "Monitor OpenSea events of the NFTs",
				Flags: []cli.Flag{
				},
			},
		},
	}
	//downloadCommand = &cli.Command{
	//	Action: download,
	//	Name:   "download",
	//	Usage:  "Download meta into mongodb",
	//	Flags: []cli.Flag{
	//		FromFlag,
	//		ToFlag,
	//	},
	//}
	botCommand = &cli.Command{
		Name:  "bot",
		Usage: "Start bot for OpenSea message dispatch",
		Subcommands: []*cli.Command{
			{
				Action:  telegramBot,
				Name:    "telegram",
				Aliases: []string{"tg"},
				Usage:   "Start a telegram bot",
			},
		},
	}
)

func updateCollection(c *cli.Context) error {
	configFile := c.String(ConfigFlag.Name)
	all := c.Bool(CollectionUpdateAllFlag.Name)
	topN := c.Int(CollectionUpdateNFlag.Name)
	daemon := c.Bool(CollectionUpdateDaemonFlag.Name)
	//from := c.Int(CollectionUpdateFromFlag.Name)
	//to := c.Int(CollectionUpdateToFlag.Name)
	//limit := c.Int(CollectionUpdateLimitFlag.Name)
	cfg := opensea.CollectionConfig{}
	if err := hs.ParseJsonConfig(configFile, &cfg); err != nil {
		return err
	}
	if topN > 0 {
		cfg.Top = topN
	}
	n := opensea.NewCollection(cfg)
	if err := n.Init(c.Context); err != nil {
		return err
	}
	defer n.Close(c.Context)

	if all {
		if err := n.UpdateAll(c.Context); err != nil {
			return err
		}
		return nil
	}
	if !daemon {
		return n.UpdateOnce(c.Context)
	}

	return n.UpdateDaemon(c.Context)
}

func listTopCollection(c *cli.Context) error {
	//address := common.HexToAddress(c.String(ContractFlag.Name))
	//from := c.Int64(FromFlag.Name)
	//to := c.Int64(ToFlag.Name)
	////log.Printf("address %s", address)
	//ec, err := ethclient.Dial("https://mainnet.infura.io/v3/e17969db9bc94e75a474b3d3c5257a75")
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//riverMen, err := erc721.NewErc721(address, ec)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//for i := from; i < to; i++ {
	//	select {
	//	case <-c.Done():
	//		return nil
	//	default:
	//		owner, err := riverMen.OwnerOf(nil, big.NewInt(int64(i)))
	//		if err != nil {
	//			log.Println(err)
	//			continue
	//		}
	//		log.Printf("%d, %s", i, owner)
	//	}
	//}
	return nil
}

func monitor(c *cli.Context) error {
	configFile := c.String(ConfigFlag.Name)
	cfg := opensea.Config{}
	if err := hs.ParseJsonConfig(configFile, &cfg); err != nil {
		return err
	}
	s := opensea.New(cfg)
	if err := s.Init(c.Context); err != nil {
		return err
	}
	defer s.Close(c.Context)
	if err := s.Monitor(c.Context); err != nil {
		return err
	}
	return nil
}

func telegramBot(c *cli.Context) error {
	//configFile := c.String(ConfigFlag.Name)
	//cfg := telegram.Config{}
	//if err := hs.ParseJsonConfig(configFile, &cfg); err != nil {
	//	return err
	//}
	//bot := telegram.New(cfg)
	//if err := bot.Init(c.Context); err != nil {
	//	return err
	//}
	//defer bot.Close(c.Context)
	//if err := bot.Serve(c.Context); err != nil {
	//	return err
	//}
	return nil
}
