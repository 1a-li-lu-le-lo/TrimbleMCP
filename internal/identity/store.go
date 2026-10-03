package identity

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/1a-li-lu-le-lo/trimblemcp/internal/fsperm"
)

// FileStore keeps a token set on disk encrypted with AES-256-GCM. The key is
// read from a separate file; plaintext storage is not supported. Production
// deployments should implement Store against a secrets manager instead.
type FileStore struct {
	Path    string
	KeyPath string
}

// storedToken mirrors Token with raw strings so the encrypted payload can be
// serialized; Token's Secret fields always marshal as "[redacted]".
type storedToken struct {
	Access  string `json:"a"`
	Refresh string `json:"r"`
	Type    string `json:"t"`
	Expiry  string `json:"e"`
	Scope   string `json:"s"`
	Next    string `json:"v"`
}

func checkPrivate(path string) error { return fsperm.CheckPrivate(path) }

func (s *FileStore) key() ([]byte, error) {
	if err := checkPrivate(s.KeyPath); err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(s.KeyPath)
	if err != nil {
		return nil, err
	}
	k, err := hex.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil || len(k) != 32 {
		return nil, errors.New("token key file must contain 64 hex characters (32 bytes)")
	}
	return k, nil
}

func (s *FileStore) aead() (cipher.AEAD, error) {
	k, err := s.key()
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(k)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// Save encrypts and atomically writes t with mode 0600.
func (s *FileStore) Save(t *Token) error {
	g, err := s.aead()
	if err != nil {
		return err
	}
	pt, err := json.Marshal(storedToken{
		Access: t.AccessToken.Reveal(), Refresh: t.RefreshToken.Reveal(), Type: t.TokenType,
		Expiry: t.Expiry.UTC().Format("2006-01-02T15:04:05Z07:00"), Scope: t.Scope,
		Next: t.NextVerifier.Reveal(),
	})
	if err != nil {
		return err
	}
	nonce := make([]byte, g.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	ct := g.Seal(nonce, nonce, pt, []byte("trimble-mcp-token-v1"))
	tmp, err := os.CreateTemp(filepath.Dir(s.Path), ".token-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write([]byte(hex.EncodeToString(ct))); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), s.Path)
}

// Load decrypts the stored token set.
func (s *FileStore) Load() (*Token, error) {
	if err := checkPrivate(s.Path); err != nil {
		return nil, err
	}
	g, err := s.aead()
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(s.Path)
	if err != nil {
		return nil, err
	}
	ct, err := hex.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil || len(ct) < g.NonceSize() {
		return nil, errors.New("token store is corrupt")
	}
	pt, err := g.Open(nil, ct[:g.NonceSize()], ct[g.NonceSize():], []byte("trimble-mcp-token-v1"))
	if err != nil {
		return nil, errors.New("token store cannot be decrypted with the configured key")
	}
	var st storedToken
	if err := json.Unmarshal(pt, &st); err != nil {
		return nil, errors.New("token store is corrupt")
	}
	t := &Token{AccessToken: Secret(st.Access), RefreshToken: Secret(st.Refresh), TokenType: st.Type, Scope: st.Scope,
		NextVerifier: Secret(st.Next)}
	if err := t.Expiry.UnmarshalText([]byte(st.Expiry)); err != nil {
		return nil, errors.New("token store is corrupt")
	}
	return t, nil
}

// FileTokenSource reads a short-lived access token from a private file on
// every call. It exists for sandbox development with a token obtained through
// a documented Trimble flow (for example the Postman callback Trimble
// preconfigures). It never refreshes.
type FileTokenSource struct{ Path string }

// Token implements connect.TokenSource.
func (f FileTokenSource) Token(context.Context) (string, error) {
	if err := checkPrivate(f.Path); err != nil {
		return "", err
	}
	b, err := os.ReadFile(f.Path)
	if err != nil {
		return "", err
	}
	tok := strings.TrimSpace(string(b))
	if tok == "" {
		return "", errors.New("access token file is empty")
	}
	return tok, nil
}

// Lock implements Locker with an exclusive lock file next to the store,
// created atomically (O_EXCL). A lock older than lockStale is presumed left
// by a crashed process and removed.
func (s *FileStore) Lock(ctx context.Context) (func(), error) {
	path := s.Path + ".lock"
	for {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			f.Close()
			return func() { os.Remove(path) }, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, err
		}
		if fi, serr := os.Stat(path); serr == nil && time.Since(fi.ModTime()) > lockStale {
			os.Remove(path)
			continue
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("waiting for the token refresh lock: %w", ctx.Err())
		case <-time.After(100 * time.Millisecond):
		}
	}
}

const lockStale = 2 * time.Minute
