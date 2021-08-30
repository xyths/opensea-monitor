package opensea

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/xyths/hs"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
	"net/http"
	"time"
)

const collName = "items"

type CollectionConfig struct {
	Mongo    hs.MongoConf
	Log      hs.LogConf
	Interval string
	Top      int
}

type Collection struct {
	cfg      CollectionConfig
	interval time.Duration

	Sugar *zap.SugaredLogger
	db    *mongo.Database
}

func NewCollection(cfg CollectionConfig) *Collection {
	return &Collection{cfg: cfg}
}

func (c *Collection) Init(ctx context.Context) error {
	l, err := hs.NewZapLogger(c.cfg.Log)
	if err != nil {
		return err
	}
	c.Sugar = l.Sugar()
	c.Sugar.Info("logger initialized")
	c.interval, err = time.ParseDuration(c.cfg.Interval)
	if err != nil {
		c.Sugar.Errorf("interval %s format error: %s", c.cfg.Interval, err)
		return err
	}
	db, err := hs.ConnectMongo(ctx, c.cfg.Mongo)
	if err != nil {
		c.Sugar.Errorf("connect mongo error: %s", err)
		return err
	}
	c.db = db
	c.Sugar.Info("database initialized")
	c.Sugar.Info("Collection initialized")
	return nil
}

func (c *Collection) Close(ctx context.Context) {
	if err := c.db.Client().Disconnect(ctx); err != nil {
		c.Sugar.Errorf("db close error: %s", err)
	}
	c.Sugar.Info("Collection closed")
}

// UpdateAll run once, update all collections and save to MongoDB.
func (c *Collection) UpdateAll(ctx context.Context) error {
	c.Sugar.Info("update all collections start")
	defer c.Sugar.Info("update all collections finish")
	offset := 0
	limit := 300
	size := limit
	for size >= limit {
		select {
		case <-ctx.Done():
			return nil
		default:
			collections, err := c.RetrieveCollections(ctx, offset, limit)
			if err != nil {
				c.Sugar.Errorf("retrieve collections error: %s", err)
				// one error, all error
				return err
			}
			size = len(collections)
			c.Sugar.Infof("Retrieve collection %d to %d", offset, offset+size)
			if err = c.saveCollections(ctx, collections); err != nil {
				c.Sugar.Errorf("save collections error: %s", err)
				return err
			}
			offset += size
		}
	}
	return nil
}

func (c *Collection) UpdateOnce(ctx context.Context) error {
	c.Sugar.Info("update all collections start")
	defer c.Sugar.Info("update all collections finish")
	return nil
}

func (c *Collection) UpdateDaemon(ctx context.Context) error {
	c.Sugar.Info("update all collections start")
	defer c.Sugar.Info("update all collections finish")
	return nil
}

// RetrieveCollections call OpenSea API to retrieve collections.
// Request like this:
// curl --request GET \
//     --url 'https://api.opensea.io/api/v1/collections?offset=0&limit=300'
func (c *Collection) RetrieveCollections(ctx context.Context, offset, limit int) ([]Item, error) {
	url := fmt.Sprintf("https://api.opensea.io/api/v1/collections?offset=%d&limit=%d", offset, limit)
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
	var responseCollections ResponseCollections
	decoder := json.NewDecoder(resp.Body)
	if err = decoder.Decode(&responseCollections); err != nil {
		return nil, err
	}
	var collections []Item
	for _, cc := range responseCollections.Collections {
		item := Item{
			Name:  cc.Name,
			Stats: cc.Stats,
		}
		l := len(cc.PrimaryAssetContracts)
		if l >= 2 {
			c.Sugar.Infof("Collection \"%s\" has %d primary_asset_contracts", cc.Name, l)
		} else if l == 1 {
			item.Address = cc.PrimaryAssetContracts[0].Address
		}

		collections = append(collections, item)
	}
	return collections, nil
}

func (c *Collection) saveCollections(ctx context.Context, collections []Item) error {
	coll := c.db.Collection(collName)
	opt := options.FindOneAndReplace().SetUpsert(true)
	for _, cc := range collections {
		//c.Sugar.Debugf("%s %s %f", cc.Address, cc.Name, cc.Stats.SevenDayVolume)
		if cc.Stats.SevenDayVolume < 100 || cc.Address == "" {
			continue
		}

		if err := coll.FindOneAndReplace(ctx,
			bson.D{{"address", cc.Address}},
			cc,
			opt,
		).Err(); err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
			c.Sugar.Errorf("Save %s error: %s", cc.Name, err)
		}
	}

	return nil
}
