package wmsqlitemodernc

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

type expiringKeyRepository struct {
	DB          *sql.DB
	StmtInsert  string
	StmtCleanUp string
	Expiration  time.Duration
}

// ExpiringKeyRepositoryConfiguration intializes the expiring key repository in [NewExpiringKeyRepository] constructor.
type ExpiringKeyRepositoryConfiguration struct {
	// Database is SQLite3 database handle.
	Database *sql.DB

	// TableName is the name of the table used to store expiring keys.
	// The
	// Defaults to "watermill_expiring_keys".
	TableName string

	// Expiration is the duration after which a key is considered expired.
	// If lower than five milliseconds, it is set to five milliseconds.
	// Defaults to one minute.
	Expiration time.Duration

	// CleanUpInterval is the interval at which expired keys are cleaned up.
	// Defaults to 15 seconds.
	CleanUpInterval time.Duration

	// CleanUpContext is the context used for cleaning up expired keys.
	// When this context is canceled, the cleanup process will end.
	// Defaults to [context.Background].
	CleanUpContext context.Context

	// CleanUpLogger tracks any problems that might emerge when cleaning up expired keys.
	// Defaults to [watermill.NewStdLogger].
	CleanUpLogger watermill.LoggerAdapter
}

// NewExpiringKeyRepository creates a repository that tracks key duplicates within a certain time frame.
// Starts a background routine to clean up expired keys. Use as a configuration option for [middleware.Deduplicator].
func NewExpiringKeyRepository(config ExpiringKeyRepositoryConfiguration) (_ middleware.ExpiringKeyRepository, err error) {
	if config.Database == nil {
		return nil, errors.New("SQLite connection is nil")
	}
	if config.TableName == "" {
		config.TableName = "watermill_expiring_keys"
	} else if err = validateTopicName(config.TableName); err != nil {
		return nil, fmt.Errorf("table name does not match topic name rules: %w", err)
	}
	if config.Expiration < time.Millisecond*5 {
		config.Expiration = time.Millisecond * 5
	}
	if config.CleanUpInterval == 0 {
		config.CleanUpInterval = 15 * time.Second
	}
	if config.CleanUpContext == nil {
		config.CleanUpContext = context.Background()
	}
	if config.CleanUpLogger == nil {
		config.CleanUpLogger = watermill.NewStdLogger(true, false)
	}

	if _, err = config.Database.ExecContext(
		config.CleanUpContext,
		`CREATE TABLE IF NOT EXISTS '`+config.TableName+`' (
			key TEXT PRIMARY KEY NOT NULL,
			expires_at INTEGER NOT NULL
		);`,
		nil); err != nil {
		return nil, fmt.Errorf("untable to create %q SQLite table: %w", config.TableName, err)
	}

	r := &expiringKeyRepository{
		DB:         config.Database,
		Expiration: config.Expiration,
	}

	r.StmtInsert = `INSERT INTO '` + config.TableName + `' (key, expires_at) VALUES (?, ?)`
	r.StmtCleanUp = `DELETE FROM '` + config.TableName + `' WHERE expires_at<?`

	go func(ctx context.Context, r *expiringKeyRepository, ticker *time.Ticker, logger watermill.LoggerAdapter) {
		defer ticker.Stop()
		var (
			err error
			t   time.Time
		)
		for {
			select {
			case <-ctx.Done():
				return
			case t = <-ticker.C:
				if err = r.CleanUp(ctx, t); err != nil {
					logger.Error("failed to clean up keys from SQLite expiring keys table", err, nil)
				} else {
					logger.Debug("cleaned up keys from SQLite expiring keys table", nil)
				}
			}
		}
	}(config.CleanUpContext, r, time.NewTicker(config.CleanUpInterval), config.CleanUpLogger)

	return r, nil
}

func (r *expiringKeyRepository) IsDuplicate(ctx context.Context, key string) (ok bool, err error) {
	if _, err = r.DB.ExecContext(ctx, r.StmtInsert, key, time.Now().Add(r.Expiration).UnixNano()); err != nil {
		var sqlError *sqlite.Error
		if errors.As(err, &sqlError) {
			if sqlError.Code() == sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY {
				return true, nil
			}
		}
		return true, err
	}
	return false, err
}

func (r *expiringKeyRepository) CleanUp(ctx context.Context, until time.Time) (err error) {
	_, err = r.DB.ExecContext(ctx, r.StmtCleanUp, until.UnixNano())
	return err
}
