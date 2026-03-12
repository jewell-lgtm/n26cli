package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/guitmz/n26"
)

var (
	ErrMFAPending  = errors.New("MFA approval pending")
	ErrMFARejected = errors.New("MFA approval rejected or failed")
)

// RequestMFAToken initiates authentication and returns the MFA token.
func RequestMFAToken(username, password, deviceToken string) (string, error) {
	data := url.Values{}
	data.Set("grant_type", "password")
	data.Set("username", username)
	data.Set("password", password)

	req, err := http.NewRequest("POST", apiURL+"/oauth2/token", strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("creating auth request: %w", err)
	}
	req.Header.Set("Authorization", "Basic bmF0aXZld2ViOg==")
	req.Header.Set("device-token", deviceToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("auth request: %w", err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("reading auth response: %w", err)
	}

	if res.StatusCode != 403 {
		return "", fmt.Errorf("auth failed (HTTP %d): %s", res.StatusCode, string(body))
	}

	var token n26.Token
	if err := json.Unmarshal(body, &token); err != nil {
		return "", fmt.Errorf("parsing MFA token: %w", err)
	}
	return token.MfaToken, nil
}

// RequestMFAChallenge sends the MFA challenge request to trigger push notification.
// This replicates n26.Token.requestMfaApproval which is unexported.
func RequestMFAChallenge(mfaToken, deviceToken string) error {
	body, err := json.Marshal(map[string]string{
		"challengeType": "oob",
		"mfaToken":      mfaToken,
	})
	if err != nil {
		return fmt.Errorf("marshaling MFA challenge: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.tech26.de/api/mfa/challenge", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("creating MFA challenge request: %w", err)
	}
	req.Header.Set("Authorization", "Basic bmF0aXZld2ViOg==")
	req.Header.Set("device-token", deviceToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/59.0.3071.86 Safari/537.36")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending MFA challenge: %w", err)
	}
	defer res.Body.Close()
	io.Copy(io.Discard, res.Body)

	if res.StatusCode != 201 {
		return fmt.Errorf("MFA challenge failed with status %d", res.StatusCode)
	}
	return nil
}

// PollMFAApproval makes a single poll attempt to complete MFA.
// Returns access/refresh tokens on success, ErrMFAPending if still waiting.
func PollMFAApproval(mfaToken, deviceToken string) (accessToken, refreshToken string, err error) {
	token := &n26.Token{MfaToken: mfaToken}
	status := token.CompleteMfaApproval(deviceToken)
	switch status {
	case 200:
		return token.AccessToken, token.RefreshToken, nil
	case 400:
		return "", "", ErrMFAPending
	default:
		return "", "", fmt.Errorf("%w: status %d", ErrMFARejected, status)
	}
}
