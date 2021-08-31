package telegram

//import (
//	"context"
//	"errors"
//	"github.com/go-telegram-bot-api/telegram-bot-api"
//	"github.com/xyths/hs"
//	"github.com/xyths/opensea-monitor/opensea"
//	"go.mongodb.org/mongo-driver/bson"
//	"go.mongodb.org/mongo-driver/mongo"
//	"go.mongodb.org/mongo-driver/mongo/options"
//	"go.uber.org/zap"
//	"time"
//)
//
//const keyLastReadTime = "lastRead"
//
//type Config struct {
//	Mongo    hs.MongoConf
//	Log      hs.LogConf
//	Bot      string // bot username
//	Token    string
//	Table    string // MongoDB collection name for bot config
//	Interval string
//}
//
//type Bot struct {
//	cfg Config
//
//	interval time.Duration
//
//	Sugar *zap.SugaredLogger
//	db    *mongo.Database
//	bot   *tgbotapi.BotAPI
//}
//
//func New(cfg Config) *Bot {
//	return &Bot{cfg: cfg}
//}
//
//func (b *Bot) Init(ctx context.Context) error {
//	l, err := hs.NewZapLogger(b.cfg.Log)
//	if err != nil {
//		return err
//	}
//	b.Sugar = l.Sugar()
//	b.Sugar.Info("logger initialized")
//	b.interval, err = time.ParseDuration(b.cfg.Interval)
//	if err != nil {
//		b.Sugar.Errorf("interval %s format error: %s", b.cfg.Interval, err)
//		return err
//	}
//	db, err := hs.ConnectMongo(ctx, b.cfg.Mongo)
//	if err != nil {
//		b.Sugar.Errorf("connect mongo error: %s", err)
//		return err
//	}
//	b.db = db
//	b.Sugar.Info("database initialized")
//	b.bot, err = tgbotapi.NewBotAPI(b.cfg.Token)
//	if err != nil {
//		b.Sugar.Errorf("New Telegram bot error: %s", err)
//		return err
//	}
//	b.Sugar.Info("Bot initialized")
//	return nil
//}
//
//func (b *Bot) Close(ctx context.Context) {
//	if err := b.db.Client().Disconnect(ctx); err != nil {
//		b.Sugar.Errorf("db close error: %s", err)
//	}
//	b.Sugar.Info("Bot closed")
//}
//
//func (b *Bot) Serve(ctx context.Context) error {
//	if err := b.doWork(ctx); err != nil {
//		b.Sugar.Errorf("doWork error: %s", err)
//	}
//	for {
//		select {
//		case <-ctx.Done():
//			return ctx.Err()
//		case <-time.After(b.interval):
//			if err := b.doWork(ctx); err != nil {
//				b.Sugar.Errorf("doWork error: %s", err)
//			}
//		}
//	}
//}
//
//func (b *Bot) doWork(ctx context.Context) error {
//	b.Sugar.Info("doWork start")
//	defer b.Sugar.Info("doWork finish")
//
//	records, err := b.getLatestEvents(ctx)
//	if err != nil {
//		return err
//	}
//	if records == nil {
//		return nil
//	}
//	//chats, err := b.getAvailableChats(ctx)
//	//if err != nil {
//	//	return err
//	//}
//	//for _, c := range chats {
//	//	if err = b.dispatch(ctx, c, records); err != nil {
//	//		b.Sugar.Errorf("dispatch error: %s", err)
//	//	}
//	//}
//	return nil
//}
//
//func (b *Bot) getLatestEvents(ctx context.Context) ([]opensea.Record, error) {
//	coll := b.db.Collection(opensea.CollEvent)
//	var now time.Time
//
//	if last, err := b.loadLastReadTime(ctx); err != nil || last == nil {
//		now = time.Now()
//	} else {
//		now = *last
//	}
//	cur, err := coll.Find(ctx,
//		bson.D{
//			{"createdAt", bson.D{
//				{"gt", now},
//			}},
//		})
//	if err != nil {
//		if errors.Is(err, mongo.ErrNoDocuments) {
//			return nil, nil
//		} else {
//			return nil, err
//		}
//	}
//	var records []opensea.Record
//	if err = cur.All(ctx, &records); err != nil {
//		return nil, err
//	}
//	return records, nil
//}
//
//func (b *Bot) loadLastReadTime(ctx context.Context) (*time.Time, error) {
//	coll := b.db.Collection(b.cfg.Table)
//	last := struct {
//		Key          string    `bson:"key"`
//		Value        time.Time `bson:"value"`
//		LastModified time.Time `bson:"lastModified"`
//	}{}
//	if err := coll.FindOne(ctx, bson.D{{"key", keyLastReadTime}}).Decode(&last); err == nil {
//		return &last.Value, nil
//	} else if errors.Is(err, mongo.ErrNoDocuments) {
//		return nil, nil
//	} else {
//		return nil, err
//	}
//}
//
//func (b *Bot) saveLastReadTime(ctx context.Context, now time.Time) error {
//	coll := b.db.Collection(b.cfg.Table)
//
//	if _, err := coll.UpdateOne(
//		ctx,
//		bson.D{
//			{"key", keyLastReadTime},
//		},
//		bson.D{
//			{"$set", bson.D{
//				{"value", now},
//			}},
//			{"$currentDate", bson.D{
//				{"lastModified", true},
//			}},
//		},
//		options.Update().SetUpsert(true),
//	); err != nil {
//		return err
//	}
//	return nil
//}
//
//func (b *Bot) getAvailableChats(ctx context.Context) ([]Configuration, error) {
//	coll := b.db.Collection(CollPreferences)
//	var chats []Configuration
//	cur, err := coll.Find(ctx,
//		bson.D{
//			{"bot", b.cfg.Bot},
//		},
//	)
//	if err != nil {
//		if errors.Is(err, mongo.ErrNoDocuments) {
//			return nil, nil
//		}
//		return nil, err
//	}
//	if err = cur.All(ctx, &chats); err != nil {
//		return nil, err
//	}
//	return chats, nil
//}
//
////func (b *Bot) dispatch(ctx context.Context, chat Configuration, records []opensea.Record) error {
////	var filter bool
////	projects := make(map[string]bool)
////	for _, p := range chat.Projects {
////		projects[common.HexToAddress(p.Address).Hex()] = true
////	}
////	if len(projects) > 0 {
////		filter = true
////	}
////	for _, r := range records {
////		if filter {
////			if _, ok := projects[r.Contract]; !ok {
////				continue
////			}
////		}
////		content := format(r, chat.Options)
////		msg := tgbotapi.NewMessage(chat.ChatId, content)
////		if _, err := b.bot.Send(msg); err != nil {
////			b.Sugar.Errorf("send message error: %s", err)
////		}
////	}
////
////	return nil
////}
//
////func (b *Bot) sendMessage(chatId int64, record op) {
////
////}
//
//func format(record opensea.Record, options Options) string {
//	//switch record.Event {
//	//case
//	//}
//	return ""
//}
