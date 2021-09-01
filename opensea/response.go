package opensea

import (
	"fmt"
	"github.com/xyths/hs/convert"
)

type ResponseEvent struct {
	Success     *bool
	AssetEvents []AssetEvent `json:"asset_events"`
}

type AssetEvent struct {
	Asset Asset `json:"asset"`
	// transfer: Mint或者是真实的Transfer
	// created: List
	// bid_entered: Bid
	// bid_withdrawn: Bid Cancel
	// successful: Sale
	// offer_entered: Offer
	EventType string `json:"event_type"`

	// used when EventType = `bid_entered` or `offer_entered`, means bid price
	BidAmount string `json:"bid_amount"`

	// used when EventType = `created`, means list price.
	EndingPrice string `json:"ending_price"`

	// used when EventType = `successful` or `bid_withdrawn`, means sale or cancel offer
	TotalPrice string `json:"total_price"`

	CreatedDate   string   `json:"created_date"`
	FromAccount   *Account `json:"from_account"`
	ToAccount     *Account `json:"to_account"`
	Owner         *Account `json:"owner"`
	Seller        *Account `json:"seller"`
	WinnerAccount *Account `json:"winner_account"`

	PaymentToken PaymentToken `json:"payment_token"`
}

const (
	EventTypeTransfer  = "transfer"
	EventTypeList      = "created"
	EventTypeBid       = "bid_entered"
	EventTypeBidCancel = "bid_withdrawn"
	EventTypeSale      = "successful"
	EventTypeOffer     = "offer_entered"
)

type Asset struct {
	TokenId       string          `json:"token_id"`
	Name          string          `json:"name"`
	AssetContract AssetContract   `json:"asset_contract"`
	Collection    AssetCollection `json:"collection"`

	ImagePreviewUrl string `json:"image_preview_url"`
}

type AssetContract struct {
	Address string `json:"address"`
	Name    string `json:"name"`
}

type AssetCollection struct {
	Name string `json:"name"`
}

type Account struct {
	User          User   `json:"user"`
	ProfileImgUrl string `json:"profile_img_url"`
	Address       string `json:"address"`
	Config        string `json:"config"`
}

func (a Account) String() string {
	addr := convert.ShortAddress(a.Address)
	if a.User.Username != "" {
		return fmt.Sprintf("%s(%s)", a.User.Username, addr)
	} else {
		return addr
	}
}

type User struct {
	Username string `json:"username"`
}

type PaymentToken struct {
	Symbol   string `json:"symbol"`
	Decimals int    `json:"decimals"`
}

// ResponseCollections is response of `/collection` API
type ResponseCollections struct {
	Collections []RawCollection `json:"collections"`
}

// RawCollection is the `collection` structure in ResponseCollections, the response of `/collection` API.
// It's different from the ResponseEvent.
type RawCollection struct {
	Name                  string
	Description           string
	PrimaryAssetContracts []RawAssetContract `json:"primary_asset_contracts"`
	Stats                 RawStat            `json:"stats"`
	CreatedDate           string             `json:"created_date"`
}

type RawStat struct {
	OneDayVolume       float64 `json:"one_day_volume" bson:"oneDayVolume"`
	OneDayChange       float64 `json:"one_day_change" bson:"oneDayChange"`
	OneDaySales        float64 `json:"one_day_sales" bson:"oneDaySales"`
	OneDayAveragePrice float64 `json:"one_day_average_price" bson:"oneDayAveragePrice"`

	SevenDayVolume       float64 `json:"seven_day_volume" bson:"sevenDayVolume"`
	SevenDayChange       float64 `json:"seven_day_change" bson:"sevenDayChange"`
	SevenDaySales        float64 `json:"seven_day_sales" bson:"sevenDaySales"`
	SevenDayAveragePrice float64 `json:"seven_day_average_price" bson:"sevenDayAveragePrice"`

	ThirtyDayVolume       float64 `json:"thirty_day_volume" bson:"thirtyDayVolume"`
	ThirtyDayChange       float64 `json:"thirty_day_change" bson:"thirtyDayChange"`
	ThirtyDaySales        float64 `json:"thirty_day_sales" bson:"thirtyDaySales"`
	ThirtyDayAveragePrice float64 `json:"thirty_day_average_price" bson:"thirtyDayAveragePrice"`

	TotalVolume float64 `json:"total_volume" bson:"totalVolume"`
	TotalSales  float64 `json:"total_sales" bson:"totalSales"`
	TotalSupply float64 `json:"total_supply" bson:"totalSupply"`

	Count        float64 `json:"count" bson:"count"`
	NumOwners    float64 `json:"num_owners" bson:"numOwners"`
	AveragePrice float64 `json:"average_price" bson:"averagePrice"`
	NumReports   float64 `json:"num_reports" bson:"numReports"`
	MarketCap    float64 `json:"market_cap" bson:"marketCap"`
	FloorPrice   float64 `json:"floor_price" bson:"floorPrice"`
}

type RawAssetContract struct {
	Address string
	Name    string
}
