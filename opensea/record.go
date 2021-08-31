package opensea

import "time"

type Record struct {
	Collection string `json:"collection"` // collection name
	Contract   string `json:"contract"`   // collection contract address
	Name       string `json:"name"`       // NFT name
	Id         string `json:"id"`
	Event      string `json:"event"`
	Price      string `json:"price"`
	From       string `json:"from"`
	To         string `json:"to"`
	Date       string `json:"date"`

	ImagePreviewUrl string `json:"imagePreviewUrl"` // for Telegram preview

	CreatedAt time.Time `json:"createdAt"`
}

const (
	EventSale      = "Sale"
	EventOffer     = "Offer"
	EventBid       = "Bid"
	EventBidCancel = "Bid Cancel"
	EventTransfer  = "Transfer"
	EventMint      = "Mint"
	EventList      = "List"
)

// Item is collection for project.
// One Item is one collection on OpenSea.
type Item struct {
	Name    string  `bson:"name"`
	Address string  `bson:"address"`
	Stats   RawStat `bson:"stats"`
	//Traits *Traits
}

// Project is a collection saved in MongoDB
type Project struct {
	Index   int    `bson:"index"`
	Name    string `bson:"name"`
	Address string `bson:"address"`
}
