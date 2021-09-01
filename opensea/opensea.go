package opensea

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/ethereum/go-ethereum/common"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/xyths/hs"
	"github.com/xyths/hs/broadcast"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
	"net/http"
	"strings"
	"time"
)

const (
	collConfig  = "config"
	collProject = "projects"
	CollEvent   = "events"

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

type TelegramConf struct {
	Bot   string // bot username
	Token string
}

type Config struct {
	Mongo    hs.MongoConf
	Log      hs.LogConf
	Interval string
	Discord  Discord
	Robots   []hs.BroadcastConf
	Telegram TelegramConf
}

type OpenSea struct {
	cfg      Config
	interval time.Duration

	Sugar *zap.SugaredLogger
	db    *mongo.Database

	discord *discordgo.Session
	robots  []broadcast.Broadcaster
	tg      *tgbotapi.BotAPI
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
	s.tg, err = tgbotapi.NewBotAPI(s.cfg.Telegram.Token)
	if err != nil {
		s.Sugar.Errorf("New Telegram bot error: %s", err)
		return err
	}
	s.Sugar.Info("Telegram bot initialized")
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
	if last != nil {
		s.Sugar.Infof("load last time: %s", last.String())
	}

	now := time.Now()
	maxDelay := time.Minute * 15
	if last == nil || last.Before(now.Add(-1*maxDelay)) {
		oldDays := now.Add(-1 * maxDelay)
		last = &oldDays
	}

	s.requestOpenSea(ctx, *last, now)
	if err = s.saveLastTime(ctx, now); err != nil {
		s.Sugar.Errorf("save last time error: %s", err)
	}
	s.Sugar.Infof("save last time: %s", now.String())
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

func (s *OpenSea) requestOpenSea(ctx context.Context, from, to time.Time) {
	topN := make(map[string]Project)
	if err := s.loadProjects(ctx, topN); err != nil {
		s.Sugar.Errorf("load topN projects error: %s", err)
	}
	for addr, _ := range topN {
		select {
		case <-ctx.Done():
			return
		default:
			if err := s.requestOpenSeaProject(ctx, addr, from, to); err != nil {
				s.Sugar.Errorf("request for single project error: %s", err)
			}
		}
	}
	return
}
func (s *OpenSea) requestOpenSeaProject(ctx context.Context, contract string, from, to time.Time) error {
	var events []Record
	offset := 0
	limit := 300
	for {
		result, size, err := s.requestOpenSeaProjectWithOffset(ctx, contract, from, to, offset, limit)
		if err != nil {
			s.Sugar.Errorf("request opensea error: %s", err)
			break
		}
		if len(result) > 0 {
			events = append(events, result...)
			s.Sugar.Debugf("records outside size = %d", len(events))
		}
		if size < limit {
			break
		}
		//s.Sugar.Infof("so many events here: offset = %d, size = %d", offset, size)
		offset += size
	}

	s.Sugar.Infof("project %s events size = %d", contract, len(events))

	if len(events) == 0 {
		return nil
	}
	chats, err := s.getAvailableChats(ctx)
	if err != nil {
		s.Sugar.Errorf("get available chats error: %s", err)
		return err
	}
	s.Sugar.Infof("chats size = %d", len(chats))
	for _, chat := range chats {
		if err = s.dispatch(ctx, chat, events); err != nil {
			s.Sugar.Errorf("dispatch events error: %s", err)
		}
	}

	return nil
}

func (s *OpenSea) requestOpenSeaProjectWithOffset(ctx context.Context, project string,
	from, to time.Time, offset, limit int) ([]Record, int, error) {
	url := fmt.Sprintf(
		"https://api.opensea.io/api/v1/events?asset_contract_address=%s&only_opensea=false&occurred_after=%d&occurred_before=%d&offset=%d&limit=%d",
		project, from.Unix(), to.Unix(), offset, limit,
	)
	s.Sugar.Infof("request: %s", url)
	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	var assetEvent ResponseEvent
	decoder := json.NewDecoder(resp.Body)
	if err = decoder.Decode(&assetEvent); err != nil {
		return nil, 0, err
	}
	if assetEvent.Success == nil {
		s.Sugar.Debugf("request success")
	} else if !(*assetEvent.Success) {
		s.Sugar.Debugf("request failed")
	}
	var result []Record
	for _, ae := range assetEvent.AssetEvents {
		result = append(result, toRecord(ae))
	}
	return result, len(assetEvent.AssetEvents), nil
}

func (s *OpenSea) saveEvent(ctx context.Context, records []interface{}) error {
	if records == nil {
		return nil
	}
	coll := s.db.Collection(CollEvent)
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
	coll := s.db.Collection(CollEvent)
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

func (s *OpenSea) getAvailableChats(ctx context.Context) ([]Configuration, error) {
	coll := s.db.Collection(CollPreferences)
	var chats []Configuration
	cur, err := coll.Find(ctx,
		bson.D{
			{"bot", s.cfg.Telegram.Bot},
		},
	)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	if err = cur.All(ctx, &chats); err != nil {
		return nil, err
	}
	return chats, nil
}

func (s *OpenSea) dispatch(ctx context.Context, chat Configuration, records []Record) error {
	var filter bool
	projects := make(map[string]bool)
	for _, p := range chat.Projects {
		projects[common.HexToAddress(p.Address).Hex()] = true
	}
	if len(projects) > 0 {
		filter = true
	}
	//s.Sugar.Infof("filter map: %v", projects)
	var count int
	for i := len(records) - 1; i >= 0; i-- {
		r := records[i]
		// send all message even if cancelled context
		if filter {
			if _, ok := projects[r.Contract]; !ok {
				s.Sugar.Debugf("event contract filtered: %s %s", r.Collection, r.Contract)
				continue
			}
		}
		content := format(r, chat.Options)
		msg := tgbotapi.NewMessage(chat.ChatId, content)
		if _, err := s.tg.Send(msg); err != nil {
			s.Sugar.Errorf("send message error: %s", err)
		}
		count++
		if count == 100 {
			time.Sleep(time.Second * 10)
			count = 0
		}
		//}
	}

	return nil
}

// format is for Telegram text message.
func format(record Record, options map[string]bool) string {
	link := options[OptionLink]
	//properties := options[telegram.OptionProperties]
	content := fmt.Sprintf("项目: %s\n名称: %s\nTokenId: %s", record.Collection, record.Name, record.Id)
	switch record.Event {
	case EventSale:
		content += fmt.Sprintf(
			` 成交(Sale)
  买家: %s
  卖家: %s
  价格: %s`,
			record.To, record.From, record.Price,
		)
	case EventOffer:
		content += fmt.Sprintf(
			` 出价(Offer)
  买家: %s
  价格: %s`,
			record.From, record.Price,
		)
	case EventBid:
		content += fmt.Sprintf(
			` 出价(Bid)
  买家: %s
  价格: %s`,
			record.From, record.Price,
		)
	case EventTypeBidCancel:
		content += fmt.Sprintf(
			` 撤销出价(Bid Cancel)
  买家: %s
  价格: %s`,
			record.From, record.Price,
		)
	case EventTransfer:
		content += fmt.Sprintf(
			` 转让(Transfer)
  发送方: %s
  接收方: %s`,
			record.From, record.To,
		)
	case EventMint:
		content += fmt.Sprintf(
			` 铸造完成 (Mint)
  接收方: %s`,
			record.To,
		)
	case EventList:
		content += fmt.Sprintf(
			` 拍卖(List)
  卖家: %s
  价格: %s`,
			record.From, record.Price,
		)
	default:
	}
	content += fmt.Sprintf("\n  时间: %s", record.Date)
	if link {
		content += fmt.Sprintf("\n地址: https://opensea.io/assets/%s/%s\n预览图片: \n%s", strings.ToLower(record.Contract), record.Id, record.ImagePreviewUrl)
	}
	return content
}
