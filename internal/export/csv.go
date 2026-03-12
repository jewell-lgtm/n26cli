package export

import (
	"encoding/csv"
	"fmt"
	"io"

	"github.com/jewell-lgtm/n26cli/internal/api"
)

func WriteCSV(w io.Writer, txns []api.Transaction) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()

	header := []string{
		"date", "amount", "currency", "merchant", "category",
		"space", "reference", "original_currency", "original_amount", "exchange_rate",
	}
	if err := cw.Write(header); err != nil {
		return err
	}

	for _, t := range txns {
		row := []string{
			t.Date.Format("2006-01-02"),
			fmt.Sprintf("%.2f", t.Amount),
			t.Currency,
			t.Merchant,
			t.Category,
			t.Space,
			t.Reference,
			t.OriginalCurrency,
			fmt.Sprintf("%.2f", t.OriginalAmount),
			fmt.Sprintf("%.4f", t.ExchangeRate),
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}

	return cw.Error()
}
