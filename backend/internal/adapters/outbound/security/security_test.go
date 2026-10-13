package security_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/dnunezd/SoftDevCloud/backend/internal/adapters/outbound/security"
	"github.com/dnunezd/SoftDevCloud/backend/internal/ports"
)

const secret = "clave-de-pruebas-de-al-menos-32-caracteres"

var (
	now    = time.Date(2026, 10, 12, 10, 0, 0, 0, time.UTC)
	claims = ports.TokenClaims{UserID: "u-1", SessionID: "s-1", ExpiresAt: now.Add(time.Hour)}
)

func clockAt(t time.Time) func() time.Time {
	return func() time.Time { return t }
}

func TestBcryptHasher(t *testing.T) {
	hasher := security.NewBcryptHasher(bcrypt.MinCost)

	hash, err := hasher.Hash("secreta123")
	require.NoError(t, err)
	assert.NotEqual(t, "secreta123", hash)

	ok, err := hasher.Compare(hash, "secreta123")
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = hasher.Compare(hash, "otra-clave")
	require.NoError(t, err)
	assert.False(t, ok)

	_, err = hasher.Compare("no-es-un-hash", "secreta123")
	assert.Error(t, err)
}

func TestJWTService_RoundTrip(t *testing.T) {
	service := security.NewJWTService(secret, clockAt(now))

	token, err := service.Issue(claims)
	require.NoError(t, err)

	parsed, err := service.Parse(token)
	require.NoError(t, err)
	assert.Equal(t, "u-1", parsed.UserID)
	assert.Equal(t, "s-1", parsed.SessionID)
	assert.True(t, parsed.ExpiresAt.Equal(claims.ExpiresAt))
}

// El token identifica al usuario y a la sesión, pero no lleva roles: los roles
// vigentes y el estado de la sesión se consultan en cada petición.
func TestJWTService_TokenCarriesSessionButNotRoles(t *testing.T) {
	token, err := security.NewJWTService(secret, clockAt(now)).Issue(claims)
	require.NoError(t, err)

	payload := jwt.MapClaims{}
	_, _, err = jwt.NewParser().ParseUnverified(token, payload)
	require.NoError(t, err)

	assert.Equal(t, "u-1", payload["sub"])
	assert.Equal(t, "s-1", payload["sid"])
	assert.NotContains(t, payload, "roles")
}

func TestJWTService_RejectsExpiredTokens(t *testing.T) {
	token, err := security.NewJWTService(secret, clockAt(now)).Issue(claims)
	require.NoError(t, err)

	_, err = security.NewJWTService(secret, clockAt(now.Add(2*time.Hour))).Parse(token)

	assert.ErrorIs(t, err, jwt.ErrTokenExpired)
}

func TestJWTService_RejectsTokensSignedWithAnotherSecret(t *testing.T) {
	token, err := security.NewJWTService("otra-clave-distinta-de-al-menos-32-caracteres", clockAt(now)).Issue(claims)
	require.NoError(t, err)

	_, err = security.NewJWTService(secret, clockAt(now)).Parse(token)

	assert.ErrorIs(t, err, jwt.ErrTokenSignatureInvalid)
}

// testClaims replica la forma de los claims para fabricar tokens maliciosos.
type testClaims struct {
	SessionID string `json:"sid,omitempty"`
	jwt.RegisteredClaims
}

func craft(t *testing.T, method jwt.SigningMethod, key any, c testClaims) string {
	t.Helper()
	token, err := jwt.NewWithClaims(method, c).SignedString(key)
	require.NoError(t, err)
	return token
}

func TestJWTService_RejectsTamperedTokens(t *testing.T) {
	service := security.NewJWTService(secret, clockAt(now))
	valid := testClaims{
		SessionID: "s-1",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "u-1",
			Issuer:    "seguimiento-monitores",
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	}
	with := func(change func(*testClaims)) testClaims {
		c := valid
		change(&c)
		return c
	}

	tests := map[string]string{
		"algoritmo none":  craft(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, valid),
		"algoritmo HS512": craft(t, jwt.SigningMethodHS512, []byte(secret), valid),
		"otro emisor":     craft(t, jwt.SigningMethodHS256, []byte(secret), with(func(c *testClaims) { c.Issuer = "otro-sistema" })),
		"sin expiración":  craft(t, jwt.SigningMethodHS256, []byte(secret), with(func(c *testClaims) { c.ExpiresAt = nil })),
		"sin sujeto":      craft(t, jwt.SigningMethodHS256, []byte(secret), with(func(c *testClaims) { c.Subject = "" })),
		"sin sesión":      craft(t, jwt.SigningMethodHS256, []byte(secret), with(func(c *testClaims) { c.SessionID = "" })),
	}
	for name, token := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := service.Parse(token)
			assert.Error(t, err)
		})
	}
}
