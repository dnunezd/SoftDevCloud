package ports

import "time"

// PasswordHasher calcula y verifica hashes de contraseñas (RNF-03).
type PasswordHasher interface {
	Hash(plain string) (string, error)
	// Compare indica si la contraseña corresponde al hash. Solo devuelve error si
	// el hash está malformado.
	Compare(hash, plain string) (bool, error)
}

// TokenClaims es el contenido de un token de acceso: quién es el usuario y a
// qué sesión pertenece el token. No incluye roles: los roles vigentes y el
// estado de la sesión se consultan en la base de datos en cada petición.
type TokenClaims struct {
	UserID    string
	SessionID string
	ExpiresAt time.Time
}

// TokenService firma y valida tokens de acceso (RF-01.1).
type TokenService interface {
	Issue(claims TokenClaims) (string, error)
	// Parse valida la firma, el emisor y la expiración del token.
	Parse(token string) (TokenClaims, error)
}
