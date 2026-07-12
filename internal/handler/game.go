package handler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"time"

	"lolipop-bot/internal/game"
	"lolipop-bot/internal/permission"

	"github.com/bwmarrin/discordgo"
)

const (
	gameExecTimeout   = 5 * time.Minute
	discordMaxContent = 2000
	successEmoji      = "✅"
	failureEmoji      = "❗️"
)

type gameActionSlashCommand struct {
	name        string
	description string
	action      game.Action
	usecase     *game.GameUsecase
	permission  *permission.Config
}

func newGameActionSlashCommand(
	name, description string,
	action game.Action,
	usecase *game.GameUsecase,
	permission *permission.Config,
) *gameActionSlashCommand {
	return &gameActionSlashCommand{
		name:        name,
		description: description,
		action:      action,
		usecase:     usecase,
		permission:  permission,
	}
}

func NewStartServerSlashCommand(usecase *game.GameUsecase, permission *permission.Config) *gameActionSlashCommand {
	return newGameActionSlashCommand("start-server", "ゲームサーバーを起動します。", game.ActionStart, usecase, permission)
}

func NewStopServerSlashCommand(usecase *game.GameUsecase, permission *permission.Config) *gameActionSlashCommand {
	return newGameActionSlashCommand("stop-server", "ゲームサーバーを停止します。", game.ActionStop, usecase, permission)
}

func NewRestartServerSlashCommand(usecase *game.GameUsecase, permission *permission.Config) *gameActionSlashCommand {
	return newGameActionSlashCommand("restart-server", "ゲームサーバーを再起動します。", game.ActionRestart, usecase, permission)
}

func NewStatusServerSlashCommand(usecase *game.GameUsecase, permission *permission.Config) *gameActionSlashCommand {
	return newGameActionSlashCommand("status-server", "ゲームサーバーの状態を確認します。", game.ActionStatus, usecase, permission)
}

func (command *gameActionSlashCommand) CreateCommand() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        command.name,
		Description: command.description,
	}
}

func (command *gameActionSlashCommand) InteractionType() discordgo.InteractionType {
	return discordgo.InteractionApplicationCommand
}

func (command *gameActionSlashCommand) InteractionID() string {
	return command.name
}

func (command *gameActionSlashCommand) MatchInteractionID(interactionID string) bool {
	return command.InteractionID() == interactionID
}

func (command *gameActionSlashCommand) Handle(session *discordgo.Session, interaction *discordgo.Interaction) error {
	userID := interaction.Member.User.ID

	if !command.permission.IsAllowed(userID) {
		log.Printf("[GAME] user %s was denied %s (mode=%s)", userID, command.action, command.permission.Mode)
		return session.InteractionRespond(interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "このコマンドを実行する権限がありません。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}

	log.Printf("[GAME] user %s requested action %s", userID, command.action)

	if err := session.InteractionRespond(interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	}); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), gameExecTimeout)
	defer cancel()

	result, err := command.usecase.Execute(ctx, command.action)

	content := ""
	if errors.Is(err, game.ErrAlreadyRunning) {
		content = "⏳ 他のコマンドが実行中です。しばらく待ってから再実行してください。"
	} else if err != nil {
		log.Printf("[GAME] action %s failed: %v", command.action, err)
		content = fmt.Sprintf("%s 実行に失敗しました: %v", failureEmoji, err)
	} else {
		content = formatResult(command.action, result)
	}

	_, editErr := session.InteractionResponseEdit(interaction, &discordgo.WebhookEdit{
		Content: &content,
	})
	return editErr
}

var actionMessages = map[game.Action]struct{ success, failure string }{
	game.ActionStart:   {successEmoji + " サーバーを起動しました。", failureEmoji + " サーバーの起動に失敗しました。"},
	game.ActionStop:    {successEmoji + " サーバーを停止しました。", failureEmoji + " サーバーの停止に失敗しました。"},
	game.ActionRestart: {successEmoji + " サーバーを再起動しました。", failureEmoji + " サーバーの再起動に失敗しました。"},
}

func formatResult(action game.Action, result *game.Result) string {
	if action == game.ActionStatus {
		if summary, ok := summarizeStatus(result.Output); ok {
			return summary
		}
		return rawResult(action, result)
	}

	messages, ok := actionMessages[action]
	if !ok {
		return rawResult(action, result)
	}

	if result.ExitCode == 0 {
		return messages.success
	}
	return truncate(messages.failure+"\n"+rawResult(action, result), discordMaxContent)
}

func rawResult(action game.Action, result *game.Result) string {
	status := successEmoji
	if result.ExitCode != 0 {
		status = fmt.Sprintf("%s (exit code: %d)", failureEmoji, result.ExitCode)
	}

	output := result.Output
	if output == "" {
		output = "(出力なし)"
	}

	header := fmt.Sprintf("%s `game %s`\n", status, action)
	body := fmt.Sprintf("```\n%s\n```", truncate(output, discordMaxContent-len(header)-8))
	return header + body
}

var activeLinePattern = regexp.MustCompile(`(?m)^\s*Active:\s*(\S+)\s*\(([^)]+)\)`)

func summarizeStatus(output string) (string, bool) {
	match := activeLinePattern.FindStringSubmatch(output)
	if match == nil {
		return "", false
	}

	state, subState := match[1], match[2]

	label := map[string]string{
		"active":       "🟢 稼働中",
		"activating":   "🟡 起動処理中",
		"deactivating": "🟡 停止処理中",
		"inactive":     "⚫ 停止中",
		"failed":       "🔴 異常停止",
	}[state]
	if label == "" {
		label = fmt.Sprintf("❔ 不明な状態(%s)", state)
	}

	return fmt.Sprintf("%s (%s)", label, subState), true
}

func truncate(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	suffix := "\n...(truncated)"
	if max <= len(suffix) {
		return s[:max]
	}
	return s[:max-len(suffix)] + suffix
}
