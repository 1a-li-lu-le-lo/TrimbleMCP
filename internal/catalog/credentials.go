package catalog

import "strings"

// notCredentials are names that end in "token" but are not secrets:
// pagination cursors, enum selectors, and Accubid's database identifier
// (a database selector returned by its Database service).
var notCredentials = map[string]bool{
	"skiptoken": true, "continuationtoken": true, "nexttoken": true, "pagetoken": true, "nextpagetoken": true,
	"deltatoken": true, "synctoken": true, "changetoken": true, "pagingtoken": true, "cursortoken": true,
	"tokentype": true, "databasetoken": true,
}

// credentialParts mark a name as carrying a secret wherever they appear.
var credentialParts = []string{
	"password", "passwd", "secret", "apikey", "privatekey", "credential", "accesstoken", "refreshtoken",
	"idtoken", "authtoken", "sessiontoken", "bearertoken",
}

// IsCredentialName reports whether a parameter or field name carries a
// credential. Names are compared case-insensitively with punctuation
// removed, so LoginPassword, client_secret, X-Api-Key and ntripPassword all
// match; pagination cursors such as skipToken do not.
func IsCredentialName(name string) bool {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	n := b.String()
	if n == "" || notCredentials[n] {
		return false
	}
	if n == "authorization" || strings.HasSuffix(n, "token") || strings.HasSuffix(n, "pwd") {
		return true
	}
	for _, p := range credentialParts {
		if strings.Contains(n, p) {
			return true
		}
	}
	return false
}
