package discord

import (
	"errors"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

type SessionConfig interface {
	Token() string
	Handlers() []any
	Slashes() []*discordgo.ApplicationCommand
}

type sessionConfig struct {
	token    string
	handlers []any
	slashes  []*discordgo.ApplicationCommand
}

func (config *sessionConfig) Token() string {
	return config.token
}

func (config *sessionConfig) Handlers() []any {
	return append([]any(nil), config.handlers...)
}

func (config *sessionConfig) Slashes() []*discordgo.ApplicationCommand {
	return append(make([]*discordgo.ApplicationCommand, 0), config.slashes...)
}

func (config *sessionConfig) validate() error {
	if config.token == "" {
		return errors.New("token is required")
	}
	if len(config.handlers) == 0 {
		return errors.New("no handlers registered")
	}
	return nil
}

type sessionConfigOption func(*sessionConfig) error

func NewSessionConfig(opts ...sessionConfigOption) (*sessionConfig, error) {
	config := &sessionConfig{}
	for _, opt := range opts {
		if err := opt(config); err != nil {
			return nil, err
		}
	}

	if err := config.validate(); err != nil {
		return nil, err
	}
	return config, nil
}

func WithToken(token string) sessionConfigOption {
	return func(config *sessionConfig) error {
		if (token) == "" {
			return fmt.Errorf("discord token is required")
		}
		config.token = token
		return nil
	}
}

func WithInteractionCreateHandler(
	handler func(*discordgo.Session, *discordgo.InteractionCreate),
) sessionConfigOption {
	return withHandler(handler)
}

func WithSlashCommand(
	command SlashCommand,
) sessionConfigOption {
	return func(config *sessionConfig) error {
		config.slashes = append(config.slashes, command.CreateCommand())
		return nil
	}
}

func withHandler(handler any) sessionConfigOption {
	return func(config *sessionConfig) error {
		config.handlers = append(config.handlers, handler)
		return nil
	}
}

type SessionManager struct {
	session *discordgo.Session
}

func (manager *SessionManager) Open(config SessionConfig) error {
	if manager.session != nil {
		_ = manager.session.Close()
		manager.session = nil
	}

	session, err := discordgo.New("Bot " + config.Token())
	if err != nil {
		return err
	}

	for _, handler := range config.Handlers() {
		session.AddHandler(handler)
	}

	if err := session.Open(); err != nil {
		return err
	}

	commands, err := session.ApplicationCommandBulkOverwrite(session.State.User.ID, "", config.Slashes())
	if err != nil {
		return err
	}

	log.Printf("[DISCORD] registered %d slash commands", len(commands))
	for _, cmd := range commands {
		log.Printf("[DISCORD]   - /%s", cmd.Name)
	}

	manager.session = session
	return nil
}

func (manager *SessionManager) Close() error {
	if manager.session == nil {
		return nil
	}

	err := manager.session.Close()
	manager.session = nil
	return err
}
