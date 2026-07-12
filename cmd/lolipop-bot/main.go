package main

import (
	"log"
	"os"
	"time"

	"lolipop-bot/internal/discord"
	"lolipop-bot/internal/game"
	"lolipop-bot/internal/handler"
	"lolipop-bot/internal/permission"
	"lolipop-bot/internal/shutdown"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load(".env")

	permissionPath := envOrDefault("PERMISSION_CONFIG_PATH", "./permissions.json")
	permConfig, err := permission.Load(permissionPath)
	if err != nil {
		log.Fatalf("[INIT] failed to load permission config: %v", err)
	}

	sshConfig := game.SSHConfig{
		Host:           os.Getenv("SSH_HOST"),
		Port:           envOrDefault("SSH_PORT", "22"),
		User:           os.Getenv("SSH_USER"),
		KeyPath:        os.Getenv("SSH_KEY_PATH"),
		PrivateKeyPEM:  os.Getenv("SSH_PRIVATE_KEY"),
		KeyPassphrase:  os.Getenv("SSH_KEY_PASSPHRASE"),
		KnownHostsPath: os.Getenv("SSH_KNOWN_HOSTS"),
		ConnectTimeout: 10 * time.Second,
	}

	gameUsecase, err := game.NewGameUsecase(sshConfig)
	if err != nil {
		log.Fatalf("[INIT] failed to initialize game usecase: %v", err)
	}

	startCmd := handler.NewStartServerSlashCommand(gameUsecase, permConfig)
	stopCmd := handler.NewStopServerSlashCommand(gameUsecase, permConfig)
	restartCmd := handler.NewRestartServerSlashCommand(gameUsecase, permConfig)
	statusCmd := handler.NewStatusServerSlashCommand(gameUsecase, permConfig)

	interactionDispatcher := &discord.InteractionDispatcher{
		Listeners: []discord.InteractionListener{
			startCmd,
			stopCmd,
			restartCmd,
			statusCmd,
		},
	}

	config, err := discord.NewSessionConfig(
		discord.WithToken(os.Getenv("DISCORD_BOT_TOKEN")),
		discord.WithInteractionCreateHandler(interactionDispatcher.OnInteractionCreate),
		discord.WithSlashCommand(startCmd),
		discord.WithSlashCommand(stopCmd),
		discord.WithSlashCommand(restartCmd),
		discord.WithSlashCommand(statusCmd),
	)
	if err != nil {
		log.Fatalf("[INIT] failed to create session config: %v", err)
	}

	var sm discord.SessionManager
	if err := sm.Open(config); err != nil {
		log.Fatalf("[INIT] failed to connect to Discord: %v", err)
	}
	defer sm.Close()

	log.Printf("[INIT] permission mode: %s", permConfig.Mode)
	log.Println("[INIT] discord bot started successfully")
	shutdown.WaitForExitSignal()
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
