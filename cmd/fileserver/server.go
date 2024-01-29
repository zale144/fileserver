package fileserver

import (
	"github.com/kelseyhightower/envconfig"
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/zale144/fileserver/internal/server/config"
	"github.com/zale144/fileserver/internal/server/database"
	"github.com/zale144/fileserver/internal/server/repository"
	"github.com/zale144/fileserver/internal/server/server"
	"github.com/zale144/fileserver/internal/server/service"
	"github.com/zale144/fileserver/internal/server/storage"
)

var ServerCmd = &cobra.Command{
	Use:   "server",
	Short: "start the file server",
	Long: `start the file server which will handle requests for file uploads, 
	downloads, and Merkle proof verification.`,
	Run: func(cmd *cobra.Command, args []string) {
		start()
	},
}

func start() {
	log, _ := zap.NewProduction()
	defer log.Sync()

	var cfg config.Config
	err := envconfig.Process("", &cfg)
	if err != nil {
		log.Fatal("Failed to process env var", zap.Error(err))
	}

	db, err := database.NewDBConnection(cfg.Database)
	if err != nil {
		log.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	if err = database.MigrateDB(db, "migrations", database.EmbedMigrations); err != nil {
		log.Fatal("Failed to migrate database", zap.Error(err))
	}

	log.Debug("Connected to database")

	repo := repository.NewFile(db)
	store, err := storage.NewFile(cfg.Storage)
	if err != nil {
		log.Fatal("Failed to create storage", zap.Error(err))
	}
	if err = store.MakeBucket(); err != nil {
		log.Fatal("Failed to create bucket", zap.Error(err))
	}

	svc := service.NewFile(repo, store, log)
	srv := server.NewServer(cfg.Server, svc, log)
	router := server.Router(srv)

	if err = srv.StartServer(router); err != nil {
		log.Fatal("Failed to start server", zap.Error(err))
	}
}
