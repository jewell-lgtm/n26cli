package api

import "fmt"

// Space represents an N26 space (sub-account).
type Space struct {
	ID        string
	Name      string
	Balance   float64
	Currency  string
	IsPrimary bool
}

// GetSpaces fetches all spaces.
func (c *Client) GetSpaces() ([]Space, error) {
	_, raw := c.inner.GetSpaces("")
	out := make([]Space, len(raw.Spaces))
	for i, s := range raw.Spaces {
		out[i] = Space{
			ID:        s.ID,
			Name:      s.Name,
			Balance:   s.Balance.AvailableBalance,
			Currency:  "EUR",
			IsPrimary: s.IsPrimary,
		}
	}
	return out, nil
}

// ResolveSpaceName maps a space ID to its name.
// Returns empty string (not error) if no matching space is found.
func (c *Client) ResolveSpaceName(spaceID string) (string, error) {
	spaces, err := c.GetSpaces()
	if err != nil {
		return "", fmt.Errorf("resolving space name: %w", err)
	}
	for _, s := range spaces {
		if s.ID == spaceID {
			return s.Name, nil
		}
	}
	return "", nil
}
