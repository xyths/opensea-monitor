package opensea

import "time"

const (
	CollPreferences = "preferences"

	OptionLink       = "link"
	OptionProperties = "properties"
)

type Configuration struct {
	Bot      string        `bson:"bot"` // bot username
	ChatId   int64         `bson:"chatId"`
	Projects []ProjectConf `bson:"projects"`
	Options  Options       `bson:"options"`
	Filter   []interface{} `bson:"filter"`
	ExpireAt time.Time     `bson:"expireAt"` // membership
}
type Options = map[string]bool

type ProjectConf struct {
	Name    string `bson:"name"`
	Address string `bson:"address"`
}

// filter the record, return true if pass.
// 1. list means `or`, map means `and`;
// 2. top level is always list;
func filter(record *Record, conf interface{}) bool {
	list, ok := conf.([]interface{})
	if ok {
		for _, f := range list { // one pass is pass
			if filter(record, f) {
				return true
			}
		}
		return false
	}
	m, ok := conf.(map[string]interface{})
	if !ok {
		return false
	}
	for k, v := range m { // all pass is pass
		switch k {
		case "price":
			if !filterPrice(record, v) {
				return false
			}
		case "properties":
			if !filterProperty(record, v) {
				return false
			}
		}
	}
	return true
}

func filterPrice(record *Record, conf interface{}) bool {
	return true
}

func filterProperty(record *Record, conf interface{}) bool {
	return true
}
