package api

import (
	"fmt"
	"time"
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
	Space            string // space name, resolved from spaceId
	SpaceID          string // raw space UUID from API
	Reference        string
}

// rawTransaction captures the full N26 API response including fields
// the guitmz/n26 library drops (notably spaceId).
type rawTransaction struct {
	ID               string  `json:"id"`
	UserID           string  `json:"userId"`
	Type             string  `json:"type"`
	Amount           float64 `json:"amount"`
	CurrencyCode     string  `json:"currencyCode"`
	OriginalAmount   float64 `json:"originalAmount,omitempty"`
	OriginalCurrency string  `json:"originalCurrency,omitempty"`
	ExchangeRate     float64 `json:"exchangeRate,omitempty"`
	MerchantName     string  `json:"merchantName,omitempty"`
	PartnerName      string  `json:"partnerName,omitempty"`
	PartnerIban      string  `json:"partnerIban,omitempty"`
	ReferenceText    string  `json:"referenceText,omitempty"`
	AccountID        string  `json:"accountId"`
	Category         string  `json:"category"`
	VisibleTS        int64   `json:"visibleTS"`
	CreatedTS        int64   `json:"createdTS"`
	// Space-related fields N26 returns but guitmz/n26 ignores
	SpaceID          *string `json:"spaceId,omitempty"`
	SpacesMisc       *string `json:"spaces,omitempty"`
}

// GetTransactions fetches transactions in the given time range directly from the
// N26 API, bypassing the guitmz/n26 library to capture spaceId.
func (c *Client) GetTransactions(from, to time.Time, limit int) ([]Transaction, error) {
	params := make(map[string]string)
	if limit > 0 {
		params["limit"] = fmt.Sprint(limit)
	}
	if !from.IsZero() && !to.IsZero() {
		params["from"] = fmt.Sprint(from.UnixMilli())
		params["to"] = fmt.Sprint(to.UnixMilli())
	}

	var raw []rawTransaction
	if err := c.get("/api/smrt/transactions", params, &raw); err != nil {
		return nil, err
	}

	return convertRawTransactions(raw), nil
}

func convertRawTransactions(raw []rawTransaction) []Transaction {
	out := make([]Transaction, len(raw))
	for i, t := range raw {
		merchant := t.MerchantName
		if merchant == "" {
			merchant = t.PartnerName
		}
		ref := t.ReferenceText
		if ref == "" && t.PartnerIban != "" {
			ref = t.PartnerIban
		}
		var spaceID string
		if t.SpaceID != nil {
			spaceID = *t.SpaceID
		}
		out[i] = Transaction{
			ID:               t.ID,
			Date:             time.UnixMilli(t.VisibleTS),
			Amount:           t.Amount,
			Currency:         t.CurrencyCode,
			OriginalAmount:   t.OriginalAmount,
			OriginalCurrency: t.OriginalCurrency,
			ExchangeRate:     t.ExchangeRate,
			Merchant:         merchant,
			Category:         t.Category,
			SpaceID:          spaceID,
			Reference:        ref,
		}
	}
	return out
}
