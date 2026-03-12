package api

import "time"

// BalanceInfo holds the current account balance.
type BalanceInfo struct {
	AvailableBalance float64
	UsableBalance    float64
	Currency         string
	AsOf             time.Time
}

// GetBalance fetches the current account balance.
func (c *Client) GetBalance() (*BalanceInfo, error) {
	_, bal := c.inner.GetBalance("")
	return &BalanceInfo{
		AvailableBalance: bal.AvailableBalance,
		UsableBalance:    bal.UsableBalance,
		Currency:         "EUR",
		AsOf:             time.Now(),
	}, nil
}
