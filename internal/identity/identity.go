// Package identity is a minimal Trimble Identity (id.trimble.com) OAuth 2.0
// client: authorization code with PKCE (S256), refresh, and revocation, plus
// token sources for adapters. Endpoints are from the issuer's published
// OpenID configuration; see docs/trimble-products/trimble-identity.md.
package identity

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/1a-li-lu-le-lo/trimblemcp/internal/errs"
)

// Issuers the bridge will talk to. Production values are verified against
// https://id.trimble.com/.well-known/openid-configuration. The staging issuer
// is referenced in Trimble's docs but its discovery document could not be
// fetched during verification; it is allowed for sandbox use only.
var Issuers = map[string]Endpoints{
	"https://id.trimble.com": {
		Authorize: "https://id.trimble.com/oauth/authorize",
		Token:     "https://id.trimble.com/oauth/token",
		Revoke:    "https://id.trimble.com/oauth/revoke",
	},
	"https://stage.id.trimble.com": {
		Authorize: "https://stage.id.trimble.com/oauth/authorize",
		Token:     "https://stage.id.trimble.com/oauth/token",
		Revoke:    "https://stage.id.trimble.com/oauth/revoke",
	},
}

// Endpoints are the OAuth endpoints for one issuer.
type Endpoints struct {
	Authorize string
	Token     string
	Revoke    string
}

// Secret is a string that never prints its value.
type Secret string

func (Secret) String() string   { return "[redacted]" }
func (Secret) GoString() string { return "[redacted]" }
func (s Secret) MarshalJSON() ([]byte, error) {
	return []byte(`"[redacted]"`), nil
}

// Reveal returns the raw value for use on the wire only.
func (s Secret) Reveal() string { return string(s) }

// Client performs OAuth exchanges.
type Client struct {
	Endpoints    Endpoints
	ClientID     string
	ClientSecret Secret // empty for public clients
	RedirectURI  string
	// Scope is space-delimited: "openid" plus the application scope issued
	// by the Trimble Developer Console.
	Scope string
	HTTP  *http.Client
}

// NewClient validates the issuer and returns a Client.
func NewClient(issuer, clientID string, secret Secret, redirect, scope string) (*Client, error) {
	ep, ok := Issuers[issuer]
	if !ok {
		return nil, errs.Newf(errs.Configuration, "issuer %q is not an allowed Trimble Identity issuer", issuer)
	}
	if clientID == "" || redirect == "" {
		return nil, errs.Newf(errs.Configuration, "client ID and redirect URI are required")
	}
	if !strings.Contains(" "+scope+" ", " openid ") {
		return nil, errs.Newf(errs.Configuration, "scope must include openid and the application scope")
	}
	return &Client{Endpoints: ep, ClientID: clientID, ClientSecret: secret, RedirectURI: redirect, Scope: scope,
		HTTP: &http.Client{Timeout: 20 * time.Second}}, nil
}

// Token is an OAuth token set.
type Token struct {
	AccessToken  Secret    `json:"access_token"`
	RefreshToken Secret    `json:"refresh_token,omitempty"`
	TokenType    string    `json:"token_type"`
	Expiry       time.Time `json:"expiry"`
	Scope        string    `json:"scope,omitempty"`
	// NextVerifier is the PKCE code_verifier whose challenge was sent with
	// the request that produced this token. Trimble Identity requires Serial
	// PKCE: the next refresh must present it along with a new challenge.
	NextVerifier Secret `json:"next_verifier,omitempty"`
}

// Valid reports whether the access token is present and not about to expire.
func (t *Token) Valid(now time.Time) bool {
	return t != nil && t.AccessToken != "" && now.Add(60*time.Second).Before(t.Expiry)
}

// PKCE holds a verifier and its S256 challenge.
type PKCE struct {
	Verifier  Secret
	Challenge string
}

// NewPKCE returns a fresh PKCE pair (RFC 7636, S256).
func NewPKCE() (PKCE, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return PKCE{}, err
	}
	v := b64(b)
	sum := sha256Sum([]byte(v))
	return PKCE{Verifier: Secret(v), Challenge: b64(sum[:])}, nil
}

// NewState returns a random state value for CSRF protection.
func NewState() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// AuthorizeURL builds the authorization request URL.
func (c *Client) AuthorizeURL(state string, p PKCE) string {
	v := url.Values{
		"response_type":         {"code"},
		"client_id":             {c.ClientID},
		"redirect_uri":          {c.RedirectURI},
		"scope":                 {c.Scope},
		"state":                 {state},
		"code_challenge":        {p.Challenge},
		"code_challenge_method": {"S256"},
	}
	return c.Endpoints.Authorize + "?" + v.Encode()
}

// Exchange trades an authorization code for tokens. Per Trimble's Serial
// PKCE rule it also sends a new code_challenge, whose verifier is kept in the
// returned token for the first refresh.
func (c *Client) Exchange(ctx context.Context, code string, p PKCE) (*Token, error) {
	next, err := NewPKCE()
	if err != nil {
		return nil, err
	}
	t, err := c.tokenRequest(ctx, url.Values{
		"grant_type":            {"authorization_code"},
		"code":                  {code},
		"redirect_uri":          {c.RedirectURI},
		"code_verifier":         {p.Verifier.Reveal()},
		"code_challenge":        {next.Challenge},
		"code_challenge_method": {"S256"},
	})
	if err != nil {
		return nil, err
	}
	t.NextVerifier = next.Verifier
	return t, nil
}

// Refresh obtains a new token set using Serial PKCE: it presents the verifier
// for the challenge sent with the previous token request, plus a new
// challenge whose verifier is kept in the returned token.
func (c *Client) Refresh(ctx context.Context, prev *Token) (*Token, error) {
	if prev == nil || prev.RefreshToken == "" {
		return nil, errs.Newf(errs.Authentication, "no refresh token; run trimblectl auth login")
	}
	if prev.NextVerifier == "" {
		return nil, errs.Newf(errs.Authentication, "the stored session predates Serial PKCE support; run trimblectl auth login")
	}
	next, err := NewPKCE()
	if err != nil {
		return nil, err
	}
	t, err := c.tokenRequest(ctx, url.Values{
		"grant_type":            {"refresh_token"},
		"refresh_token":         {prev.RefreshToken.Reveal()},
		"code_verifier":         {prev.NextVerifier.Reveal()},
		"code_challenge":        {next.Challenge},
		"code_challenge_method": {"S256"},
	})
	if err != nil {
		return nil, err
	}
	if t.RefreshToken == "" {
		// Some issuers do not rotate; keep the existing refresh token.
		t.RefreshToken = prev.RefreshToken
	}
	t.NextVerifier = next.Verifier
	return t, nil
}

// Revoke revokes a refresh token (documented as Basic client auth with
// token and token_type_hint).
func (c *Client) Revoke(ctx context.Context, refresh Secret) error {
	form := url.Values{"token": {refresh.Reveal()}, "token_type_hint": {"refresh_token"}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoints.Revoke, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(url.QueryEscape(c.ClientID), url.QueryEscape(c.ClientSecret.Reveal()))
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return errs.Wrap(errs.UpstreamUnavailable, err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<16))
	if resp.StatusCode/100 != 2 {
		return errs.Newf(errs.Authentication, "token revocation failed with status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) tokenRequest(ctx context.Context, form url.Values) (*Token, error) {
	// Trimble documents client_id on both token and refresh requests.
	form.Set("client_id", c.ClientID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoints.Token, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	if c.ClientSecret != "" {
		req.SetBasicAuth(url.QueryEscape(c.ClientID), url.QueryEscape(c.ClientSecret.Reveal()))
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, errs.Wrap(errs.UpstreamUnavailable, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, errs.Wrap(errs.UpstreamUnavailable, err)
	}
	if resp.StatusCode/100 != 2 {
		// The body may echo request details; keep only the OAuth error code.
		var e struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(body, &e)
		return nil, errs.Newf(errs.Authentication, "token request rejected (status %d, %s)", resp.StatusCode, sanitizeCode(e.Error))
	}
	var w struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int64  `json:"expires_in"`
		Scope        string `json:"scope"`
	}
	if err := json.Unmarshal(body, &w); err != nil || w.AccessToken == "" {
		return nil, errs.Newf(errs.UpstreamMalformed, "token response is malformed")
	}
	if !strings.EqualFold(w.TokenType, "bearer") {
		return nil, errs.Newf(errs.UpstreamMalformed, "unexpected token type")
	}
	if w.ExpiresIn <= 0 {
		return nil, errs.Newf(errs.UpstreamMalformed, "token response has no expires_in")
	}
	return &Token{
		AccessToken: Secret(w.AccessToken), RefreshToken: Secret(w.RefreshToken), TokenType: "Bearer",
		Expiry: time.Now().UTC().Add(time.Duration(w.ExpiresIn) * time.Second), Scope: w.Scope,
	}, nil
}

func sanitizeCode(s string) string {
	for _, r := range s {
		if !(r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z') {
			return "unknown_error"
		}
	}
	if s == "" || len(s) > 64 {
		return "unknown_error"
	}
	return s
}

// ---- token sources ----

// Store persists a token set.
type Store interface {
	Load() (*Token, error)
	Save(*Token) error
}

// RefreshingSource returns a valid access token, refreshing and persisting
// (rotating) the refresh token as needed. It is safe for concurrent use.
type RefreshingSource struct {
	Client *Client
	Store  Store
	mu     sync.Mutex
	cur    *Token
	now    func() time.Time
}

// Token implements connect.TokenSource.
func (s *RefreshingSource) Token(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	if s.now != nil {
		now = s.now()
	}
	if s.cur == nil {
		t, err := s.Store.Load()
		if err != nil {
			return "", errs.Wrap(errs.Authentication, fmt.Errorf("no stored Trimble Identity session: %w", err))
		}
		s.cur = t
	}
	if s.cur.Valid(now) {
		return s.cur.AccessToken.Reveal(), nil
	}
	if s.cur.RefreshToken == "" {
		return "", errs.Newf(errs.Authentication, "the Trimble Identity session expired; run trimblectl auth login")
	}
	t, err := s.Client.Refresh(ctx, s.cur)
	if err != nil {
		return "", err
	}
	if err := s.Store.Save(t); err != nil {
		return "", errs.Wrap(errs.Configuration, err)
	}
	s.cur = t
	return t.AccessToken.Reveal(), nil
}

func sha256Sum(b []byte) [32]byte { return sha256.Sum256(b) }
func b64(b []byte) string         { return base64.RawURLEncoding.EncodeToString(b) }
