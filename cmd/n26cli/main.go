package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/jewell-lgtm/n26cli/internal/api"
	"github.com/jewell-lgtm/n26cli/internal/auth"
	"github.com/jewell-lgtm/n26cli/internal/export"
)

var format string

func main() {
	root := &cobra.Command{
		Use:   "n26cli",
		Short: "N26 banking CLI — interactive auth for humans, structured output for agents",
		Long: `n26cli wraps the N26 API with two modes of operation:

  INTERACTIVE (human):
    n26cli login      Launch TUI to authenticate with email + password + 2FA

  STRUCTURED (agent-safe):
    n26cli status     Check session validity (JSON to stdout)
    n26cli balance    Print account balance (JSON to stdout)
    n26cli spaces     List N26 Spaces with balances (JSON to stdout)
    n26cli transactions  Fetch & export transactions (CSV/JSON/MT940 to files)

The login command requires a human to approve 2FA on their phone.
All other commands are non-interactive and produce structured output.

Session is stored at ~/.config/n26cli/session.json (0600 permissions).
No credentials are ever persisted — only the bearer token.

Exit codes:
  0  Success
  1  Auth error (expired/missing session)
  2  API error (N26 returned an error)
  3  Export error (file I/O failure)`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(loginCmd())
	root.AddCommand(statusCmd())
	root.AddCommand(balanceCmd())
	root.AddCommand(spacesCmd())
	root.AddCommand(transactionsCmd())
	root.AddCommand(configCmd())

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

// --- login ---

func loginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Authenticate with N26 (interactive TUI — requires human)",
		Long: `Launch an interactive TUI to authenticate with N26.

Steps:
  1. Enter email and password (or set N26_USERNAME / N26_PASSWORD env vars)
  2. A device token is generated/reused from ~/.config/n26cli/device.json
  3. N26 sends a 2FA push notification to your phone — approve it
  4. Bearer token is saved to ~/.config/n26cli/session.json

This command CANNOT be run by an agent — it requires interactive input
and phone-based 2FA approval. Agents should call 'n26cli status' to
check if a valid session exists before making data requests.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := auth.RunLoginTUI()
			if err != nil {
				return exitError(1, "auth_failed", err.Error())
			}
			return nil
		},
	}
}

// --- status ---

func statusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Check session validity (agent-safe)",
		Long: `Print current authentication status as JSON.

Output fields:
  authenticated     bool    Whether a valid session exists
  expires_at        string  ISO 8601 expiry timestamp (if authenticated)
  minutes_remaining int     Minutes until session expires

Use this before any data command to decide if re-authentication is needed.
If authenticated is false, prompt the human to run 'n26cli login'.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			s := auth.LoadSession()
			out := map[string]interface{}{
				"authenticated": s.IsValid(),
			}
			if s != nil {
				out["expires_at"] = s.ExpiresAt.Format(time.RFC3339)
				out["minutes_remaining"] = s.MinutesRemaining()
			} else {
				out["minutes_remaining"] = 0
			}
			return printJSON(out)
		},
	}
	addFormatFlag(cmd)
	return cmd
}

// --- balance ---

func balanceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "balance",
		Short: "Print account balance (agent-safe)",
		Long: `Fetch and print the current N26 account balance.

Output fields:
  available_balance  float64  Available balance in EUR
  usable_balance     float64  Usable balance in EUR
  currency           string   Always "EUR"
  as_of              string   ISO 8601 timestamp of the query

Requires a valid session — run 'n26cli login' first if expired.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := authenticatedClient()
			if err != nil {
				return err
			}
			bal, err := client.GetBalance()
			if err != nil {
				return exitError(2, "api_error", err.Error())
			}
			return printJSON(bal)
		},
	}
	addFormatFlag(cmd)
	return cmd
}

// --- spaces ---

func spacesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "spaces",
		Short: "List N26 Spaces with balances (agent-safe)",
		Long: `Fetch and print all N26 Spaces (sub-accounts) with their balances.

Output: JSON array of objects with fields:
  id        string   Space UUID
  name      string   Space name (e.g., "Main", "Savings", "Travel")
  balance   float64  Current balance in EUR
  currency  string   Always "EUR"

Requires a valid session — run 'n26cli login' first if expired.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := authenticatedClient()
			if err != nil {
				return err
			}
			spaces, err := client.GetSpaces()
			if err != nil {
				return exitError(2, "api_error", err.Error())
			}
			return printJSON(spaces)
		},
	}
	addFormatFlag(cmd)
	return cmd
}

// --- transactions ---

func transactionsCmd() *cobra.Command {
	var (
		from         string
		to           string
		groupBySpace bool
		outputDir    string
		limit        int
	)

	cmd := &cobra.Command{
		Use:   "transactions",
		Short: "Fetch & export transactions (agent-safe)",
		Long: `Fetch N26 transactions and export to CSV, JSON, or MT940 files.

By default fetches the last 30 days. Use --from and --to for custom ranges.

When --group-by-space is set:
  • Transactions are grouped by N26 Space name
  • One file per Space: {space-slug}_{from}_{to}.{ext}
  • Transactions with no Space go into main_{from}_{to}.{ext}
  • An additional _all_{from}_{to}.{ext} contains everything

CSV columns:
  date, amount, currency, merchant, category, space, reference,
  original_currency, original_amount, exchange_rate

Files are written to --output-dir (default: current directory).
Set a persistent default with: n26cli config set output-dir .DONOTCOMMIT/data/n26

Requires a valid session — run 'n26cli login' first if expired.`,
		Example: `  # Last 30 days as CSV
  n26cli transactions

  # Custom date range, grouped by space, to data dir
  n26cli transactions \
    --from 2026-02-01 --to 2026-03-12 \
    --group-by-space \
    --output-dir .DONOTCOMMIT/data/n26

  # JSON format, last 7 days
  n26cli transactions --from 2026-03-05 --format json

  # MT940 for accounting software import
  n26cli transactions --from 2026-01-01 --format mt940`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Use config default for output-dir if flag wasn't explicitly set
			if !cmd.Flags().Changed("output-dir") {
				if cfg := auth.LoadConfig(); cfg.OutputDir != "" {
					outputDir = cfg.OutputDir
				}
			}

			client, err := authenticatedClient()
			if err != nil {
				return err
			}

			fromTime, toTime, err := parseDateRange(from, to)
			if err != nil {
				return exitError(3, "invalid_dates", err.Error())
			}

			txns, err := client.GetTransactions(fromTime, toTime, limit)
			if err != nil {
				return exitError(2, "api_error", err.Error())
			}

			// Always resolve space names from spaceId
			spaces, err := client.GetSpaces()
			if err != nil {
				return exitError(2, "api_error", "fetching spaces: "+err.Error())
			}
			spaceMap := make(map[string]string)
			var primarySpaceName string
			for _, sp := range spaces {
				spaceMap[sp.ID] = sp.Name
				if sp.IsPrimary {
					primarySpaceName = sp.Name
				}
			}
			if primarySpaceName == "" {
				primarySpaceName = "Main"
			}
			for i, t := range txns {
				if t.SpaceID != "" {
					if name, ok := spaceMap[t.SpaceID]; ok {
						txns[i].Space = name
					} else {
						txns[i].Space = t.SpaceID // fallback to raw ID
					}
				} else {
					txns[i].Space = primarySpaceName
				}
			}

			if err := os.MkdirAll(outputDir, 0755); err != nil {
				return exitError(3, "export_error", "creating output dir: "+err.Error())
			}

			fromStr := fromTime.Format("2006-01-02")
			toStr := toTime.Format("2006-01-02")

			if groupBySpace {
				grouped := groupTransactions(txns)
				for spaceName, spaceTxns := range grouped {
					slug := slugify(spaceName)
					filename := fmt.Sprintf("%s_%s_%s.%s", slug, fromStr, toStr, formatExt())
					if err := writeExport(filepath.Join(outputDir, filename), spaceTxns); err != nil {
						return exitError(3, "export_error", err.Error())
					}
					fmt.Fprintf(os.Stderr, "Wrote %d transactions to %s\n", len(spaceTxns), filename)
				}
			}

			// Always write _all file
			allFilename := fmt.Sprintf("_all_%s_%s.%s", fromStr, toStr, formatExt())
			if err := writeExport(filepath.Join(outputDir, allFilename), txns); err != nil {
				return exitError(3, "export_error", err.Error())
			}
			fmt.Fprintf(os.Stderr, "Wrote %d transactions to %s\n", len(txns), allFilename)

			// Print summary to stdout as JSON
			return printJSON(map[string]interface{}{
				"total_transactions": len(txns),
				"from":              fromStr,
				"to":                toStr,
				"output_dir":        outputDir,
				"format":            format,
			})
		},
	}

	thirtyDaysAgo := time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	today := time.Now().Format("2006-01-02")

	cmd.Flags().StringVar(&from, "from", thirtyDaysAgo, "Start date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&to, "to", today, "End date (YYYY-MM-DD)")
	cmd.Flags().BoolVar(&groupBySpace, "group-by-space", false, "Split output files by N26 Space")
	cmd.Flags().StringVar(&outputDir, "output-dir", ".", "Directory to write output files")
	cmd.Flags().IntVar(&limit, "limit", 0, "Max transactions to return (0 = all)")
	addFormatFlag(cmd)

	return cmd
}

// --- config ---

func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "View or set persistent configuration",
		Long: `View or set persistent CLI configuration.

Config is stored at ~/.config/n26cli/config.json.

Available settings:
  output_dir    Default directory for transaction exports

With no subcommand, prints current config as JSON.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return printJSON(auth.LoadConfig())
		},
	}

	setCmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a config value",
		Long: `Set a persistent config value.

Available keys:
  output-dir    Default directory for transaction exports
                (overridden by --output-dir flag on transactions command)

Examples:
  n26cli config set output-dir .DONOTCOMMIT/data/n26
  n26cli config set output-dir ~/finances/n26`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := auth.LoadConfig()
			switch args[0] {
			case "output-dir":
				cfg.OutputDir = args[1]
			default:
				return exitError(3, "unknown_key", fmt.Sprintf("unknown config key %q (available: output-dir)", args[0]))
			}
			if err := cfg.Save(); err != nil {
				return exitError(3, "config_error", err.Error())
			}
			return printJSON(cfg)
		},
	}

	getCmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Get a config value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := auth.LoadConfig()
			switch args[0] {
			case "output-dir":
				fmt.Println(cfg.OutputDir)
			default:
				return exitError(3, "unknown_key", fmt.Sprintf("unknown config key %q", args[0]))
			}
			return nil
		},
	}

	cmd.AddCommand(setCmd, getCmd)
	return cmd
}

// --- helpers ---

func addFormatFlag(cmd *cobra.Command) {
	cmd.Flags().StringVar(&format, "format", "json", "Output format: json, csv, mt940, or table")
}

func formatExt() string {
	switch format {
	case "csv":
		return "csv"
	case "mt940":
		return "sta"
	default:
		return "json"
	}
}

func authenticatedClient() (*api.Client, error) {
	s := auth.LoadSession()
	if !s.IsValid() {
		return nil, exitError(1, "session_expired",
			"No valid session. Run `n26cli login` to authenticate.")
	}
	return api.NewClientFromToken(s.AccessToken), nil
}

func printJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func parseDateRange(from, to string) (time.Time, time.Time, error) {
	layout := "2006-01-02"
	f, err := time.Parse(layout, from)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid --from date %q: %w", from, err)
	}
	t, err := time.Parse(layout, to)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid --to date %q: %w", to, err)
	}
	// Set to end of day
	t = t.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	return f, t, nil
}

func groupTransactions(txns []api.Transaction) map[string][]api.Transaction {
	groups := make(map[string][]api.Transaction)
	for _, t := range txns {
		space := t.Space
		if space == "" {
			space = "main"
		}
		groups[space] = append(groups[space], t)
	}
	return groups
}

func slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	// Remove non-alphanumeric chars except hyphens
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func writeExport(path string, txns []api.Transaction) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating %s: %w", path, err)
	}
	defer f.Close()

	switch format {
	case "csv":
		return export.WriteCSV(f, txns)
	case "mt940":
		return export.WriteMT940(f, txns, "", time.Now())
	default:
		return export.WriteJSON(f, txns)
	}
}

type cliError struct {
	ErrorCode string `json:"error"`
	Message   string `json:"message"`
	Code      int    `json:"code"`
}

func (e *cliError) Error() string {
	data, _ := json.MarshalIndent(e, "", "  ")
	fmt.Fprintln(os.Stderr, string(data))
	return e.Message
}

func exitError(code int, errCode, msg string) *cliError {
	return &cliError{ErrorCode: errCode, Message: msg, Code: code}
}
