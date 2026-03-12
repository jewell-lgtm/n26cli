package auth

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/huh"


	"github.com/jewell-lgtm/n26cli/internal/api"
	"github.com/jewell-lgtm/n26cli/internal/tui"
)

// RunLoginTUI runs the interactive login flow.
// Returns the session on success.
func RunLoginTUI() (*Session, error) {
	// Check existing session
	if s := LoadSession(); s.IsValid() {
		fmt.Println(tui.SuccessMessage.Render(
			fmt.Sprintf("✓ Already authenticated. Session valid for %d more minutes.", s.MinutesRemaining()),
		))
		return s, nil
	}

	// Load/create device token
	deviceToken, isNew, err := LoadOrCreateDeviceToken()
	if err != nil {
		return nil, fmt.Errorf("device token: %w", err)
	}
	if isNew {
		fmt.Println(tui.InfoBox.Render(
			"New device token generated.\n" +
				"You may need to approve this device in your N26 app.\n\n" +
				"Token: " + deviceToken,
		))
		fmt.Println()
	}

	// Credential form
	email := os.Getenv("N26_USERNAME")
	password := os.Getenv("N26_PASSWORD")

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Email").
				Value(&email).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("email is required")
					}
					return nil
				}),
			huh.NewInput().
				Title("Password").
				EchoMode(huh.EchoModePassword).
				Value(&password).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("password is required")
					}
					return nil
				}),
		),
	).WithTheme(tui.N26Theme())

	fmt.Println(tui.Title.Render("N26 Login"))

	if err := form.Run(); err != nil {
		return nil, fmt.Errorf("login form: %w", err)
	}

	// Request MFA token
	fmt.Println()
	mfaToken, err := api.RequestMFAToken(email, password, deviceToken)
	if err != nil {
		return nil, fmt.Errorf("MFA token request: %w", err)
	}

	// Request MFA challenge (push to phone)
	if err := api.RequestMFAChallenge(mfaToken, deviceToken); err != nil {
		return nil, fmt.Errorf("MFA challenge: %w", err)
	}

	// Poll for approval
	fmt.Println(tui.Subtitle.Render("Approve the login request on your phone"))
	fmt.Println()

	timeout := 60 * time.Second
	interval := 5 * time.Second
	deadline := time.Now().Add(timeout)

	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return nil, fmt.Errorf("2FA approval timed out after %s", timeout)
		}

		fmt.Printf("\r  ◐ Waiting for 2FA approval... (%ds remaining)  ", int(remaining.Seconds()))

		accessToken, refreshToken, err := api.PollMFAApproval(mfaToken, deviceToken)
		if err == nil {
			fmt.Println()
			// N26 tokens typically last ~15 minutes
			session := &Session{
				AccessToken:  accessToken,
				RefreshToken: refreshToken,
				ExpiresAt:    time.Now().Add(15 * time.Minute),
			}
			if err := session.Save(); err != nil {
				return nil, fmt.Errorf("saving session: %w", err)
			}
			fmt.Println(tui.SuccessMessage.Render(
				fmt.Sprintf("✓ Authenticated. Session valid until %s.", session.ExpiresAt.Format("15:04")),
			))
			return session, nil
		}

		time.Sleep(interval)
	}
}
