package anthropic

import (
	"fmt"
	"os"
	"time"

	"github.com/joshsgoldstein/lazyburn/internal/models"
	"github.com/joshsgoldstein/lazyburn/internal/parser"
)

// Provider reads Claude Code sessions from ~/.claude/projects/.
type Provider struct {
	claudeDir string
}

// New returns an Anthropic provider pointed at the default Claude data directory.
func New() *Provider {
	home, _ := os.UserHomeDir()
	return &Provider{claudeDir: fmt.Sprintf("%s/.claude", home)}
}

func (p *Provider) Name() string {
	return "Anthropic (Claude Code)"
}

func (p *Provider) ParseSessions(since, until time.Time, pathFilter string) ([]models.Session, error) {
	return parser.ParseAllSessions(p.claudeDir, since, until, pathFilter)
}
