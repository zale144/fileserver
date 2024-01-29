package config

import (
	"github.com/zale144/fileserver/internal/server/database"
	"github.com/zale144/fileserver/internal/server/server"
	"github.com/zale144/fileserver/internal/server/storage"
)

// Config is the configuration for the server.
type Config struct {
	Database database.Config
	Storage  storage.Config
	Server   server.Config
}
