package api

import (
	"fmt"
	"time"

	"github.com/guitmz/n26"
)

// Transaction is a clean representation of an n26 transaction.
type Transaction struct {
	ID               string
	Date             time.Time
	Amount           float64
	Currency         string
	OriginalAmount   float64
	OriginalCurrency string
	ExchangeRate     float64
	Merchant         string
	Category         string
	Space            string // resolved later via ResolveSpaceName
	Reference        string
}

// GetTransactions fetches transactions in the given time range.
// Pass zero time for from/to to use server defaults. Limit 0 means no limit.
func (c *Client) GetTransactions(from, to time.Time, limit int) ([]Transaction, error) {
	var fromTS, toTS n26.TimeStamp
	if !from.IsZero() {
		fromTS = n26.TimeStamp{Time: from}
	}
	if !to.IsZero() {
		toTS = n26.TimeStamp{Time: to}
	}

	limitStr := fmt.Sprint(limit)
	if limit <= 0 {
		limitStr = ""
	}

	raw, err := c.inner.GetTransactions(fromTS, toTS, limitStr)
	if err != nil {
		return nil, err
	}
	return convertTransactions(raw), nil
}

func convertTransactions(raw *n26.Transactions) []Transaction {
	if raw == nil {
		return nil
	}
	out := make([]Transaction, len(*raw))
	for i, t := range *raw {
		merchant := t.MerchantName
		if merchant == "" {
			merchant = t.PartnerName
		}
		ref := t.ReferenceText
		if ref == "" && t.PartnerIban != "" {
			ref = t.PartnerIban
		}
		out[i] = Transaction{
			ID:               t.ID,
			Date:             t.VisibleTS.Time,
			Amount:           t.Amount,
			Currency:         t.CurrencyCode,
			OriginalAmount:   t.OriginalAmount,
			OriginalCurrency: t.OriginalCurrency,
			ExchangeRate:     t.ExchangeRate,
			Merchant:         merchant,
			Category:         t.Category,
			Reference:        ref,
		}
	}
	return out
}
