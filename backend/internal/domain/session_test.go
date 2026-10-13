package domain_test

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"

	"github.com/dnunezd/SoftDevCloud/backend/internal/domain"
)

func TestNewSession(t *testing.T) {
	now := time.Date(2026, 10, 12, 8, 0, 0, 0, time.UTC)

	t.Run("vence una duración después de su creación", func(t *testing.T) {
		s := domain.NewSession("u-1", "Mozilla/5.0", "10.0.0.7", now, 8*time.Hour)

		assert.Equal(t, "u-1", s.UserID)
		assert.Equal(t, "Mozilla/5.0", s.UserAgent)
		assert.Equal(t, "10.0.0.7", s.IP)
		assert.Equal(t, now.Add(8*time.Hour), s.ExpiresAt)
	})

	t.Run("normaliza agentes de usuario que no son UTF-8 válido", func(t *testing.T) {
		latin1 := "Firefox (port\xe1til)" // "á" en Latin-1, como la envía una terminal de Windows

		s := domain.NewSession("u-1", latin1, "", now, time.Hour)

		assert.True(t, utf8.ValidString(s.UserAgent))
		assert.Equal(t, "Firefox (port�til)", s.UserAgent)
	})

	t.Run("recorta agentes de usuario demasiado largos sin cortar caracteres", func(t *testing.T) {
		s := domain.NewSession("u-1", strings.Repeat("ñ", 300), "", now, time.Hour)

		assert.Equal(t, domain.MaxUserAgentLength, utf8.RuneCountInString(s.UserAgent))
		assert.True(t, utf8.ValidString(s.UserAgent))
	})
}
