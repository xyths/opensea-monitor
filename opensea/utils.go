package opensea

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
	"time"
)

func toRecord(ae AssetEvent) Record {
	r := Record{
		Collection:      ae.Asset.Collection.Name,
		Contract:        common.HexToAddress(ae.Asset.AssetContract.Address).Hex(),
		Name:            ae.Asset.Name,
		Id:              ae.Asset.TokenId,
		From:            ae.FromAccount.String(),
		Date:            toBeijingTime(ae.CreatedDate),
		CreatedAt:       time.Now(),
		ImagePreviewUrl: ae.Asset.ImagePreviewUrl,
	}
	switch ae.EventType {
	case EventTypeTransfer:
		if ae.FromAccount.Address == "0x0000000000000000000000000000000000000000" {
			r.Event = EventMint
		} else {
			r.Event = EventTransfer
		}
		// Mint 和 Transfer 都没有价格需要展示
	case EventTypeList:
		r.Event = EventList
		r.Price = toEther(ae.EndingPrice, ae.PaymentToken)
	case EventTypeBid:
		r.Event = EventBid
		r.Price = toEther(ae.BidAmount, ae.PaymentToken)
	case EventTypeBidCancel:
		r.Event = EventBidCancel
		r.Price = toEther(ae.TotalPrice, ae.PaymentToken)
	case EventTypeSale:
		r.Event = EventSale
		r.Price = toEther(ae.TotalPrice, ae.PaymentToken)
		r.From = ae.Seller.String()
		r.To = ae.WinnerAccount.String()
	case EventTypeOffer:
		r.Event = EventOffer
		r.Price = toEther(ae.BidAmount, ae.PaymentToken)
	default:
		r.Event = ae.EventType
	}

	return r
}

// "2021-08-28T09:44:43.664713"
func toBeijingTime(date string) string {
	//secondsEastOfUTC := int((8 * time.Hour).Seconds())
	//beijing := time.FixedZone("Beijing Time", secondsEastOfUTC)
	layout := "2006-01-02T15:04:05.999999"
	t, err := time.Parse(layout, date)
	if err != nil {
		return date
	}
	onlyTime := "15:04:05"
	return t.Local().Format(onlyTime)
}

func toEther(price string, payment PaymentToken) string {
	unit := decimal.New(1, int32(payment.Decimals))
	d, err := decimal.NewFromString(price)
	if err != nil {
		return price
	}
	ret := fmt.Sprintf("%s %s", d.Div(unit).String(), payment.Symbol)
	return ret
}
