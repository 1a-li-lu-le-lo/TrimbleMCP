package identity

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPKCEChallengeRFC7636Vector(t *testing.T) {
	// RFC 7636 Appendix B.
	p, err := NewPKCE()
	if err != nil || len(p.Verifier) != 43 {
		t.Fatalf("verifier length %d %v", len(p.Verifier), err)
	}
	const v = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	c := challengeFor(v)
	if c != "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM" {
		t.Fatalf("challenge %s", c)
	}
}

// challengeFor uses the same derivation helpers as NewPKCE.
func challengeFor(v string) string {
	sum := sha256Sum([]byte(v))
	return b64(sum[:])
}

func TestSecretNeverPrints(t *testing.T) {
	s := Secret("hunter2")
	b, _ := json.Marshal(struct{ S Secret }{s})
	out := fmt.Sprintf("%v %s %+v %#v %s", s, s, s, s, b)
	if strings.Contains(out, "hunter2") {
		t.Fatalf("secret leaked: %s", out)
	}
	tok := Token{AccessToken: "at-123", RefreshToken: "rt-456"}
	b, _ = json.Marshal(tok)
	if strings.Contains(string(b)+fmt.Sprintf("%+v", tok), "at-123") {
		t.Fatal("token leaked")
	}
}

func TestNewClientValidation(t *testing.T) {
	if _, err := NewClient("https://evil.example", "c", "", "http://127.0.0.1/cb", "openid app"); err == nil {
		t.Fatal("unknown issuer must be rejected")
	}
	if _, err := NewClient("https://id.trimble.com", "c", "", "http://127.0.0.1/cb", "app"); err == nil {
		t.Fatal("scope without openid must be rejected")
	}
	c, err := NewClient("https://id.trimble.com", "c", "", "http://127.0.0.1:8765/callback", "openid app")
	if err != nil {
		t.Fatal(err)
	}
	p, _ := NewPKCE()
	u, _ := url.Parse(c.AuthorizeURL("st", p))
	q := u.Query()
	if u.Host != "id.trimble.com" || q.Get("code_challenge_method") != "S256" || q.Get("state") != "st" || q.Get("code_challenge") != p.Challenge {
		t.Fatalf("authorize url %s", u)
	}
	if strings.Contains(u.String(), p.Verifier.Reveal()) {
		t.Fatal("verifier must not appear in the authorize URL")
	}
}

func testClient(t *testing.T, h http.HandlerFunc) *Client {
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return &Client{Endpoints: Endpoints{Token: srv.URL + "/token", Revoke: srv.URL + "/revoke"},
		ClientID: "cid", ClientSecret: "csecret", RedirectURI: "http://127.0.0.1/cb", Scope: "openid app", HTTP: srv.Client()}
}

func TestExchangeAndRefresh(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		u, p, ok := r.BasicAuth()
		if !ok || u != "cid" || p != "csecret" {
			t.Error("client auth missing")
		}
		switch r.Form.Get("grant_type") {
		case "authorization_code":
			if r.Form.Get("code_verifier") == "" || r.Form.Get("code") != "the-code" {
				t.Error("pkce verifier or code missing")
			}
			w.Write([]byte(`{"access_token":"at1","refresh_token":"rt1","token_type":"bearer","expires_in":3600}`))
		case "refresh_token":
			w.Write([]byte(`{"access_token":"at2","token_type":"Bearer","expires_in":3600}`))
		}
	})
	p, _ := NewPKCE()
	tok, err := c.Exchange(context.Background(), "the-code", p)
	if err != nil || tok.AccessToken.Reveal() != "at1" || !tok.Valid(time.Now()) {
		t.Fatalf("%+v %v", tok, err)
	}
	tok2, err := c.Refresh(context.Background(), tok.RefreshToken)
	if err != nil || tok2.AccessToken.Reveal() != "at2" || tok2.RefreshToken.Reveal() != "rt1" {
		t.Fatalf("refresh: %v", err)
	}
}

func TestTokenErrorDoesNotEchoBody(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		w.Write([]byte(`{"error":"invalid_grant","error_description":"code abc for user bob@corp"}`))
	})
	_, err := c.Refresh(context.Background(), "rt")
	if err == nil || strings.Contains(err.Error(), "bob@corp") || !strings.Contains(err.Error(), "invalid_grant") {
		t.Fatalf("err %v", err)
	}
}

func TestMalformedTokenResponses(t *testing.T) {
	for _, body := range []string{`{}`, `{"access_token":"a","token_type":"mac","expires_in":1}`, `{"access_token":"a","token_type":"bearer"}`, `nope`} {
		c := testClient(t, func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(body)) })
		if _, err := c.Refresh(context.Background(), "rt"); err == nil {
			t.Errorf("accepted malformed body %s", body)
		}
	}
}

func writeKey(t *testing.T, dir string, mode os.FileMode) string {
	p := filepath.Join(dir, "key")
	if err := os.WriteFile(p, []byte(strings.Repeat("ab", 32)), mode); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestFileStoreRoundTripAndPermissions(t *testing.T) {
	dir := t.TempDir()
	st := &FileStore{Path: filepath.Join(dir, "tok"), KeyPath: writeKey(t, dir, 0o600)}
	in := &Token{AccessToken: "at", RefreshToken: "rt", TokenType: "Bearer", Expiry: time.Now().Add(time.Hour).UTC().Truncate(time.Second)}
	if err := st.Save(in); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(st.Path)
	if strings.Contains(string(raw), `"r"`) || strings.Contains(string(raw), "Bearer") {
		t.Fatal("token store is not encrypted")
	}
	fi, _ := os.Stat(st.Path)
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("mode %v", fi.Mode().Perm())
	}
	out, err := st.Load()
	if err != nil || out.RefreshToken.Reveal() != "rt" || !out.Expiry.Equal(in.Expiry) {
		t.Fatalf("%+v %v", out, err)
	}
	// Wrong key.
	os.WriteFile(st.KeyPath, []byte(strings.Repeat("cd", 32)), 0o600)
	if _, err := st.Load(); err == nil {
		t.Fatal("wrong key accepted")
	}
	// World-readable key refused.
	loose := &FileStore{Path: st.Path, KeyPath: writeKey(t, t.TempDir(), 0o644)}
	if err := loose.Save(in); err == nil {
		t.Fatal("world-readable key accepted")
	}
}

type memStore struct {
	t     *Token
	saves int
}

func (m *memStore) Load() (*Token, error) { return m.t, nil }
func (m *memStore) Save(t *Token) error   { m.t = t; m.saves++; return nil }

func TestRefreshingSourceRotates(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"access_token":"fresh","refresh_token":"rt2","token_type":"Bearer","expires_in":3600}`))
	})
	ms := &memStore{t: &Token{AccessToken: "stale", RefreshToken: "rt1", Expiry: time.Now().Add(-time.Minute)}}
	src := &RefreshingSource{Client: c, Store: ms}
	got, err := src.Token(context.Background())
	if err != nil || got != "fresh" || ms.saves != 1 || ms.t.RefreshToken.Reveal() != "rt2" {
		t.Fatalf("%q %v saves=%d", got, err, ms.saves)
	}
	got, _ = src.Token(context.Background())
	if got != "fresh" || ms.saves != 1 {
		t.Fatal("valid token should be reused without refresh")
	}
}
