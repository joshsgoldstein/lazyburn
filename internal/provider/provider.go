package provider

import (
	"time"

	"github.com/joshsgoldstein/lazyburn/internal/models"
)

// Provider reads session data from an AI coding tool's data directory.
// Implement this interface to add support for a new tool or platform.
type Provider interface {
	Name() string
	ParseSessions(since, until time.Time, pathFilter string) ([]models.Session, error)
}
