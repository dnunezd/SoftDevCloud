package security

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// BcryptHasher implementa ports.PasswordHasher con bcrypt.
type BcryptHasher struct {
	cost int
}

// NewBcryptHasher crea el hasher con el costo indicado (BCRYPT_COST). Un costo
// mayor es más seguro, pero cada inicio de sesión consume más CPU.
func NewBcryptHasher(cost int) *BcryptHasher {
	return &BcryptHasher{cost: cost}
}

// Hash calcula el hash bcrypt de la contraseña.
func (h *BcryptHasher) Hash(plain string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), h.cost)
	if err != nil {
		return "", fmt.Errorf("bcrypt: %w", err)
	}
	return string(hash), nil
}

// Compare indica si la contraseña corresponde al hash.
func (h *BcryptHasher) Compare(hash, plain string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("bcrypt: %w", err)
	}
	return true, nil
}
