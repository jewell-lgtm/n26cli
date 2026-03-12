package export

import (
	"encoding/json"
	"io"

	"github.com/jewell-lgtm/n26cli/internal/api"
)

type jsonTransaction struct {
	ID               string  `json:"id"`
	Date             string  `json:"date"`
	Amount           float64 `json:"amount"`
	Currency         string  `json:"currency"`
	OriginalAmount   float64 `json:"original_amount"`
	OriginalCurrency string  `json:"original_currency"`
	ExchangeRate     float64 `json:"exchange_rate"`
	Merchant         string  `json:"merchant"`
	Category         string  `json:"category"`
	Space            string  `json:"space"`
	Reference        string  `json:"reference"`
}

func WriteJSON(w io.Writer, txns []api.Transaction) error {
	out := make([]jsonTransaction, len(txns))
	for i, t := range txns {
		out[i] = jsonTransaction{
			ID:               t.ID,
			Date:             t.Date.Format("2006-01-02"),
			Amount:           t.Amount,
			Currency:         t.Currency,
			OriginalAmount:   t.OriginalAmount,
			OriginalCurrency: t.OriginalCurrency,
			ExchangeRate:     t.ExchangeRate,
			Merchant:         t.Merchant,
			Category:         t.Category,
			Space:            t.Space,
			Reference:        t.Reference,
		}
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
