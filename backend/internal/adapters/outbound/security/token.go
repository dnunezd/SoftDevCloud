package security

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/dnunezd/SoftDevCloud/backend/internal/ports"
)

// tokenIssuer identifica a este sistema como emisor de los tokens.
const tokenIssuer = "seguimiento-monitores"

// JWTService implementa ports.TokenService con JWT firmados con HS256 (ADR-08).
// El token identifica al usuario (sub) y a su sesión (sid); la vigencia de la
// sesión y los roles se verifican en la base de datos en cada petición.
type JWTService struct {
	secret []byte
	now    func() time.Time
}

// NewJWTService crea el servicio. now permite fijar el reloj en las pruebas.
func NewJWTService(secret string, now func() time.Time) *JWTService {
	return &JWTService{secret: []byte(secret), now: now}
}

type claims struct {
	SessionID string `json:"sid"`
	jwt.RegisteredClaims
}

// Issue firma un token para la sesión. Vence junto con la sesión.
func (s *JWTService) Issue(tc ports.TokenClaims) (string, error) {
	c := claims{
		SessionID: tc.SessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   tc.UserID,
			Issuer:    tokenIssuer,
			IssuedAt:  jwt.NewNumericDate(s.now()),
			ExpiresAt: jwt.NewNumericDate(tc.ExpiresAt),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("firmar token: %w", err)
	}
	return token, nil
}

// Parse valida firma, algoritmo, emisor y expiración, y devuelve el contenido.
func (s *JWTService) Parse(token string) (ports.TokenClaims, error) {
	var c claims
	_, err := jwt.ParseWithClaims(token, &c,
		func(*jwt.Token) (any, error) { return s.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(tokenIssuer),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(s.now),
	)
	if err != nil {
		return ports.TokenClaims{}, fmt.Errorf("token inválido: %w", err)
	}
	if c.Subject == "" || c.SessionID == "" {
		return ports.TokenClaims{}, errors.New("token inválido: sin usuario o sin sesión")
	}
	return ports.TokenClaims{UserID: c.Subject, SessionID: c.SessionID, ExpiresAt: c.ExpiresAt.Time}, nil
}
