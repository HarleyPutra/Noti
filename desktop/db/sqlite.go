package db

import (
	"database/sql"
	"desktop/models"
    "fmt"
	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func Init(path string) error {
	var err error
	DB, err = sql.Open("sqlite3", path+"?_journal_mode=WAL")
	if err != nil {
		return err
	}
	return runMigrations()
}

func runMigrations() error {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS todos (
			id           TEXT PRIMARY KEY,
			user_id      TEXT NOT NULL,
			title        TEXT NOT NULL,
			notes        TEXT DEFAULT '',
			done         INTEGER DEFAULT 0,
			created_at   INTEGER NOT NULL,
			updated_at   INTEGER NOT NULL,
			deleted      INTEGER DEFAULT 0,
			synced       INTEGER DEFAULT 0,
			version      INTEGER DEFAULT 1,
			vector_clock TEXT DEFAULT '{}'
		);
		CREATE TABLE IF NOT EXISTS sync_meta (
			key   TEXT PRIMARY KEY,
			value TEXT
		);
	`)
	return err
}

func GetTodos(userID string) ([]models.Todo, error) {
	rows, err := DB.Query(
		`SELECT id,user_id,title,notes,done,created_at,updated_at,
		        deleted,synced,version,vector_clock
		 FROM todos WHERE user_id = ? AND deleted = 0
		 ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var todos []models.Todo
	for rows.Next() {
		var t models.Todo
		var done, deleted, synced int
		err := rows.Scan(
			&t.ID, &t.UserID, &t.Title, &t.Notes, &done,
			&t.CreatedAt, &t.UpdatedAt, &deleted, &synced,
			&t.Version, &t.VectorClock,
		)
		if err != nil {
			return nil, err
		}
		t.Done = done == 1
		t.Deleted = deleted == 1
		t.Synced = synced == 1
		todos = append(todos, t)
	}
	if todos == nil {
		todos = []models.Todo{}
	}
	return todos, nil
}

func UpsertTodo(t models.Todo) error {
	done, deleted, synced := 0, 0, 0
	if t.Done    { done = 1 }
	if t.Deleted { deleted = 1 }
	if t.Synced  { synced = 1 }

	_, err := DB.Exec(`
		INSERT INTO todos
			(id,user_id,title,notes,done,created_at,updated_at,
			 deleted,synced,version,vector_clock)
		VALUES (?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET
			title        = excluded.title,
			notes        = excluded.notes,
			done         = excluded.done,
			updated_at   = excluded.updated_at,
			deleted      = excluded.deleted,
			synced       = excluded.synced,
			version      = excluded.version,
			vector_clock = excluded.vector_clock
	`, t.ID, t.UserID, t.Title, t.Notes, done,
		t.CreatedAt, t.UpdatedAt, deleted, synced,
		t.Version, t.VectorClock,
	)
	return err
}

func GetUnsynced(userID string) ([]models.Todo, error) {
	rows, err := DB.Query(
		`SELECT id,user_id,title,notes,done,created_at,updated_at,
		        deleted,synced,version,vector_clock
		 FROM todos WHERE user_id = ? AND synced = 0`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var todos []models.Todo
	for rows.Next() {
		var t models.Todo
		var done, deleted, synced int
		rows.Scan(
			&t.ID, &t.UserID, &t.Title, &t.Notes, &done,
			&t.CreatedAt, &t.UpdatedAt, &deleted, &synced,
			&t.Version, &t.VectorClock,
		)
		t.Done = done == 1
		t.Deleted = deleted == 1
		t.Synced = synced == 1
		todos = append(todos, t)
	}
	if todos == nil {
		todos = []models.Todo{}
	}
	return todos, nil
}

func MarkSynced(id string) error {
	_, err := DB.Exec(`UPDATE todos SET synced = 1 WHERE id = ?`, id)
	return err
}

func GetLastSyncTime() int64 {
	var val string
	DB.QueryRow(`SELECT value FROM sync_meta WHERE key = 'last_sync'`).Scan(&val)
	if val == "" {
		return 0
	}
	var t int64
	fmt.Sscanf(val, "%d", &t)
	return t
}

func SetLastSyncTime(t int64) {
	DB.Exec(`
		INSERT INTO sync_meta (key,value) VALUES ('last_sync',?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, fmt.Sprintf("%d", t))
}