package opensea

import (
	"fmt"
	"github.com/shopspring/decimal"
	"github.com/xyths/hs/convert"
	"math"
	"time"
)

func toRecord(ae AssetEvent) Record {
	r := Record{
		Collection: ae.Asset.Collection.Name,
		Name:       ae.Asset.Name,
		Id:         ae.Asset.TokenId,
		From:       toName(ae.FromAccount),
		Date:       toBeijingTime(ae.CreatedDate),
		CreatedAt:  time.Now(),
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
	case EventTypeOffer:
		r.Event = EventOffer
		r.Price = toEther(ae.BidAmount, ae.PaymentToken)
	default:
		r.Event = ae.EventType
	}

	return r
}

func toName(account Account) string {
	addr := convert.ShortAddress(account.Address)
	if account.User.Username != "" {
		return fmt.Sprintf("%s(%s)", account.User.Username, addr)
	} else {
		return addr
	}
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
	//t = t.In(time.Local)
	return t.Local().String()
}

func toEther(price string, payment PaymentToken) string {
	unit := math.Pow(10, float64(payment.Decimals))
	d, err := decimal.NewFromString(price)
	if err != nil {
		return price
	}
	ret := fmt.Sprintf("%s %s", d.Div(decimal.NewFromFloat(unit)).String(), payment.Symbol)
	return ret
}
