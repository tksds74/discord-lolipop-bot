package discord

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

type SlashCommand interface {
	CreateCommand() *discordgo.ApplicationCommand
}

type InteractionApplicationListener interface {
	SlashCommand
	InteractionListener
}

type InteractionListener interface {
	InteractionType() discordgo.InteractionType
	InteractionID() string
	MatchInteractionID(InteractionID string) bool
	Handle(session *discordgo.Session, interaction *discordgo.Interaction) error
}

type InteractionDispatcher struct {
	Listeners []InteractionListener
}

func (dispatcher *InteractionDispatcher) OnInteractionCreate(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	for _, listener := range dispatcher.Listeners {
		if listener.InteractionType() != interaction.Type {
			continue
		}

		if want := listener.InteractionID(); want != "" {
			got := ""
			if interaction.Type == discordgo.InteractionApplicationCommand {
				got = interaction.ApplicationCommandData().Name
			}

			if !listener.MatchInteractionID(got) {
				continue
			}
		}

		if err := listener.Handle(session, interaction.Interaction); err != nil {
			log.Printf("[DISCORD] failed to handle interaction: %v", err)
		}
	}
}
