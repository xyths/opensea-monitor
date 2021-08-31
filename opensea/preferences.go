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
	ExpireAt time.Time     `bson:"expireAt"` // membership
}
type Options = map[string]bool

type ProjectConf struct {
	Name    string `bson:"name"`
	Address string `bson:"address"`
}
