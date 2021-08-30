package opensea

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/ethereum/go-ethereum/common"
	"github.com/xyths/hs"
	"github.com/xyths/hs/broadcast"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
	"net/http"
	"time"
)

const (
	collConfig  = "config"
	collProject = "projects"
	collEvent   = "events"

	keyLastUpdateTime = "lastUpdateTime"

	expireIndexName = "eventExpireIndex"
)

type Discord struct {
	Token         string
	SaleChannels  []string `json:"saleChannels"`
	OfferChannels []string `json:"offerChannels"`
	BidChannels   []string `json:"bidChannels"`
	OtherChannels []string `json:"otherChannels"`
	RobotChannels []string `json:"robotChannels"`
}

type Config struct {
	Mongo    hs.MongoConf
	Log      hs.LogConf
	Interval string
	Discord  Discord
	Robots   []hs.BroadcastConf
}

type OpenSea struct {
	cfg      Config
	interval time.Duration

	Sugar *zap.SugaredLogger
	db    *mongo.Database

	discord *discordgo.Session
	robots  []broadcast.Broadcaster
}

func New(cfg Config) *OpenSea {
	return &OpenSea{cfg: cfg}
}

func (s *OpenSea) Init(ctx context.Context) error {
	l, err := hs.NewZapLogger(s.cfg.Log)
	if err != nil {
		return err
	}
	s.Sugar = l.Sugar()
	s.Sugar.Info("logger initialized")
	s.interval, err = time.ParseDuration(s.cfg.Interval)
	if err != nil {
		s.Sugar.Errorf("interval %s format error: %s", s.cfg.Interval, err)
		return err
	}
	db, err := hs.ConnectMongo(ctx, s.cfg.Mongo)
	if err != nil {
		s.Sugar.Errorf("connect mongo error: %s", err)
		return err
	}
	s.db = db
	if err = s.initIndex(ctx); err != nil {
		s.Sugar.Errorf("init index error: %s", err)
		return err
	}
	s.Sugar.Info("database initialized")
	//s.discord, err = discordgo.New("Bot " + s.cfg.Discord.Token)
	//if err != nil {
	//	s.Sugar.Errorf("discord bot init error: %s", err)
	//	return err
	//}
	//for _, conf := range s.cfg.Robots {
	//	s.robots = append(s.robots, broadcast.New(conf))
	//}
	s.Sugar.Info("OpenSea initialized")
	return nil
}

func (s *OpenSea) Close(ctx context.Context) {
	//if err := s.discord.Close(); err != nil {
	//	s.Sugar.Errorf("discord close error: %s", err)
	//}
	if err := s.db.Client().Disconnect(ctx); err != nil {
		s.Sugar.Errorf("db close error: %s", err)
	}
	s.Sugar.Info("OpenSea closed")
}

func (s *OpenSea) Monitor(ctx context.Context) error {
	if err := s.doWork(ctx); err != nil {
		s.Sugar.Errorf("doWork error: %s", err)
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(s.interval):
			if err := s.doWork(ctx); err != nil {
				s.Sugar.Errorf("doWork error: %s", err)
			}
		}
	}
}

func (s *OpenSea) doWork(ctx context.Context) error {
	s.Sugar.Info("doWork start")
	defer s.Sugar.Info("doWork finish")
	last, err := s.loadLastTime(ctx)
	if err != nil {
		//s.Sugar.Errorf("load last update time error: %s", err)
		return err
	}
	now := time.Now()
	if last == nil {
		yesterday := now.Add(-1 * time.Hour * 24)
		last = &yesterday
	}
	events, err := s.requestOpenSea(ctx, *last, now)
	if err != nil {
		return err
	}
	if err = s.saveEvent(ctx, events); err != nil {
		return err
	}
	if err = s.saveLastTime(ctx, now); err != nil {
		return err
	}
	return nil
}

func (s *OpenSea) loadLastTime(ctx context.Context) (*time.Time, error) {
	coll := s.db.Collection(collConfig)
	last := struct {
		Key          string    `bson:"key"`
		Value        time.Time `bson:"value"`
		LastModified time.Time `bson:"lastModified"`
	}{}
	if err := coll.FindOne(ctx, bson.D{{"key", keyLastUpdateTime}}).Decode(&last); err == nil {
		return &last.Value, nil
	} else if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	} else {
		return nil, err
	}
}

func (s *OpenSea) saveLastTime(ctx context.Context, now time.Time) error {
	coll := s.db.Collection(collConfig)

	if _, err := coll.UpdateOne(
		ctx,
		bson.D{
			{"key", keyLastUpdateTime},
		},
		bson.D{
			{"$set", bson.D{
				{"value", now},
			}},
			{"$currentDate", bson.D{
				{"lastModified", true},
			}},
		},
		options.Update().SetUpsert(true),
	); err != nil {
		return err
	}
	return nil
}

func (s *OpenSea) requestOpenSea(ctx context.Context, from, to time.Time) ([]interface{}, error) {
	url := fmt.Sprintf("https://api.opensea.io/api/v1/events?only_opensea=false&occurred_after=%d&occurred_before=%d", from.Unix(), to.Unix())
	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var assetEvent ResponseEvent
	decoder := json.NewDecoder(resp.Body)
	if err = decoder.Decode(&assetEvent); err != nil {
		return nil, err
	}
	topN := make(map[string]Project)
	_ = s.loadProjects(ctx, topN)
	var records []interface{}
	for _, ae := range assetEvent.AssetEvents {
		if _, ok := topN[common.HexToAddress(ae.Asset.AssetContract.Address).Hex()]; !ok {
			continue
		}
		records = append(records, toRecord(ae))
	}
	return records, nil
}

func (s *OpenSea) saveEvent(ctx context.Context, records []interface{}) error {
	if records == nil {
		return nil
	}
	coll := s.db.Collection(collEvent)
	_, err := coll.InsertMany(ctx, records)
	return err
}

func (s *OpenSea) loadProjects(ctx context.Context, projects map[string]Project) error {
	coll := s.db.Collection(collProject)
	cur, err := coll.Find(ctx, bson.D{})
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil
		}
		return err
	}
	var records []Project
	if err = cur.All(ctx, &records); err != nil {
		return err
	}
	for _, p := range records {
		p.Address = common.HexToAddress(p.Address).Hex()
		projects[p.Address] = p
	}
	return nil
}

func (s *OpenSea) initIndex(ctx context.Context) error {
	// list index first
	coll := s.db.Collection(collEvent)
	indexView := coll.Indexes()
	cursor, err := indexView.List(ctx, options.ListIndexes().SetMaxTime(time.Second*2))
	if err != nil {
		return err
	}
	var indexes []bson.M
	if err = cursor.All(ctx, &indexes); err != nil {
		return err
	}
	for _, index := range indexes {
		if index["name"] == expireIndexName {
			//log.Println("index already exist")
			return nil
		}
	}
	index := mongo.IndexModel{
		Keys:    bson.D{{"createdAt", 1}},
		Options: options.Index().SetExpireAfterSeconds(60 * 10).SetName(expireIndexName), // 10 min
	}
	name, err := indexView.CreateOne(ctx, index)
	if err != nil {
		return err
	}
	s.Sugar.Infof("create index %s", name)
	return nil
}
