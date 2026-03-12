package export

import (
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"github.com/jewell-lgtm/n26/internal/api"
)

func WriteMT940(w io.Writer, txns []api.Transaction, iban string, statementDate time.Time) error {
	var closing float64
	for _, t := range txns {
		closing += t.Amount
	}

	currency := "EUR"
	if len(txns) > 0 {
		currency = txns[0].Currency
	}

	var b strings.Builder

	// Transaction reference
	b.WriteString(fmt.Sprintf(":20:N26-%s\r\n", statementDate.Format("20060102")))
	// Account identification
	b.WriteString(fmt.Sprintf(":25:%s\r\n", iban))
	// Statement number
	b.WriteString(":28C:1/1\r\n")
	// Opening balance: assume zero
	b.WriteString(fmt.Sprintf(":60F:%s%s%s%s\r\n",
		"C", statementDate.Format("060102"), currency, fmtAmount(0)))

	for _, t := range txns {
		dc := "C"
		if t.Amount < 0 {
			dc = "D"
		}
		ref := t.Reference
		if ref == "" {
			ref = t.Merchant
		}
		if ref == "" {
			ref = "NONREF"
		}
		// :61: value date, entry date, debit/credit, amount, transaction type, reference
		b.WriteString(fmt.Sprintf(":61:%s%s%s%sN%s\r\n",
			t.Date.Format("060102"),
			t.Date.Format("0102"),
			dc,
			fmtAmount(t.Amount),
			ref,
		))
		// :86: information to account owner
		detail := t.Merchant
		if t.Category != "" {
			detail += " | " + t.Category
		}
		if detail == "" {
			detail = ref
		}
		b.WriteString(fmt.Sprintf(":86:%s\r\n", detail))
	}

	// Closing balance
	closingDC := "C"
	if closing < 0 {
		closingDC = "D"
	}
	b.WriteString(fmt.Sprintf(":62F:%s%s%s%s\r\n",
		closingDC, statementDate.Format("060102"), currency, fmtAmount(closing)))

	_, err := io.WriteString(w, b.String())
	return err
}

// fmtAmount formats an amount for MT940: no sign, comma as decimal separator.
func fmtAmount(v float64) string {
	s := fmt.Sprintf("%.2f", math.Abs(v))
	return strings.Replace(s, ".", ",", 1)
}
