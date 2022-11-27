package migration

import (
	"database/sql"
	"embed"
	"flag"
	"github.com/jmoiron/sqlx"
	"github.com/khvh/gwf/pkg/logger"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
	"github.com/rs/zerolog/log"
	"os"
)

// Init migrations
func Init(migrations embed.FS, flags *flag.FlagSet, dbType, dsn string) {
	logger.Init(os.Getenv("DEV") != "")

	err := flags.Parse(os.Args[1:])
	if err != nil {
		log.Panic().Err(err).Send()
	}

	args := flags.Args()

	db, err := sqlx.Open(dbType, dsn)
	if err != nil {
		log.Panic().Err(err).Send()
	}

	if len(args) > 0 {
		migrate(args[0], migrations, db.DB)
	}
}

func migrate(command string, migrations embed.FS, db *sql.DB) {
	goose.SetBaseFS(migrations)

	if err := goose.SetDialect("postgres"); err != nil {
		log.Panic().Err(err).Send()
	}

	if err := goose.Run(command, db, "migrations"); err != nil {
		log.Panic().Err(err).Send()
	}
}