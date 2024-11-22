//go:build cgo

package core

import (
	"database/sql"
	"log"

	"github.com/mutecomm/go-sqlcipher/v4"
	"github.com/pocketbase/dbx"
)

func init() {
	// Registers the sqlite3 driver with a ConnectHook so that we can
	// initialize the default PRAGMAs.
	//
	// Note 1: we don't define the PRAGMA as part of the dsn string
	// because not all pragmas are available.
	//
	// Note 2: the busy_timeout pragma must be first because
	// the connection needs to be set to block on busy before WAL mode
	// is set in case it hasn't been already set by another connection.
	sql.Register("pb_sqlite3",
		&sqlite3.SQLiteDriver{
			ConnectHook: func(conn *sqlite3.SQLiteConn) error {
				// Set up encryption and other PRAGMAs for SQLCipher
				_, err := conn.Exec(`
						PRAGMA key = 'your_encryption_key';  -- Set the encryption key here
						PRAGMA busy_timeout       = 10000;
						PRAGMA journal_mode       = WAL;
						PRAGMA journal_size_limit = 200000000;
						PRAGMA synchronous        = NORMAL;
						PRAGMA foreign_keys       = ON;
						PRAGMA temp_store         = MEMORY;
						PRAGMA cache_size         = -16000;
					`, nil)

				return err
			},
		},
	)

	// Register the builder function for the SQLCipher driver
	dbx.BuilderFuncMap["pb_sqlite3"] = dbx.BuilderFuncMap["sqlite3"]
}

func connectDB(dbPath string) (*dbx.DB, error) {
	// Define the encryption key pragma (replace "your-secret-key" with a real secret key)
	db, err := dbx.Open("sqlite3", dbPath+"?_pragma_key=your-secret-key&_pragma_cipher_page_size=4096")
	if err != nil {
		return nil, err
	}

	// Initialize the database pragmas for performance
	if err := initPragmas(db); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func initPragmas(db *dbx.DB) error {
	// List of recommended PRAGMA settings for SQLCipher/SQLite
	pragmas := []string{
		"PRAGMA busy_timeout = 10000;",           // Timeout on locked database
		"PRAGMA journal_mode = WAL;",             // Use Write-Ahead Logging for concurrency
		"PRAGMA journal_size_limit = 200000000;", // Limit journal size
		"PRAGMA synchronous = NORMAL;",           // Set synchronous to NORMAL for a good balance of reliability/performance
		"PRAGMA foreign_keys = ON;",              // Enforce foreign key constraints
		"PRAGMA temp_store = MEMORY;",            // Store temp files in memory to improve speed
		"PRAGMA cache_size = -16000;",            // Negative value implies cache size in KB, so -16000 sets 16MB cache
	}

	// Execute each PRAGMA statement
	for _, pragma := range pragmas {
		_, err := db.NewQuery(pragma).Execute()
		if err != nil {
			log.Printf("Failed to execute PRAGMA %s: %v", pragma, err)
			return err
		}
	}

	return nil
}
