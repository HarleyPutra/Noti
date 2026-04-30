package db

import (
	"database/sql"
	"fmt"
	"noti/models"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func Init(path string) error {
	var err error
	DB, err = sql.Open("sqlite", path+"?_journal_mode=WAL")
	if err != nil {
		return err
	}
	return runMigrations()
}

func runMigrations() error {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS notes (
			id           TEXT PRIMARY KEY,
			user_id      TEXT NOT NULL,
			title        TEXT DEFAULT 'New Note',
			content      TEXT DEFAULT '',
			mode         TEXT DEFAULT 'list',
			color        TEXT DEFAULT '#6b3f3f',
			bg_color     TEXT DEFAULT '#e8e0d5',
			pinned       INTEGER DEFAULT 0,
			width        INTEGER DEFAULT 400,
			height       INTEGER DEFAULT 500,
			pos_x        INTEGER DEFAULT 100,
			pos_y        INTEGER DEFAULT 100,
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

func GetNotes(userID string) ([]models.Note, error) {
	rows, err := DB.Query(
		`SELECT id,user_id,title,content,mode,color,pinned,
		        width,height,pos_x,pos_y,created_at,updated_at,
		        deleted,synced,version,vector_clock
		 FROM notes WHERE user_id = ? AND deleted = 0
		 ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []models.Note
	for rows.Next() {
		var n models.Note
		var pinned, deleted, synced int
		err := rows.Scan(
			&n.ID, &n.UserID, &n.Title, &n.Content, &n.Mode,
			&n.Color, &pinned, &n.Width, &n.Height,
			&n.PosX, &n.PosY, &n.CreatedAt, &n.UpdatedAt,
			&deleted, &synced, &n.Version, &n.VectorClock,
		)
		if err != nil {
			return nil, err
		}
		n.Pinned = pinned == 1
		n.Deleted = deleted == 1
		n.Synced = synced == 1
		notes = append(notes, n)
	}
	if notes == nil {
		notes = []models.Note{}
	}
	return notes, nil
}

func UpsertNote(n models.Note) error {
	pinned, deleted, synced := 0, 0, 0
	if n.Pinned  { pinned = 1 }
	if n.Deleted { deleted = 1 }
	if n.Synced  { synced = 1 }

	_, err := DB.Exec(`
		INSERT INTO notes
			(id, user_id, title, content, mode, color, bg_color, pinned, 
			 width, height, pos_x, pos_y, created_at, updated_at, 
			 deleted, synced, version, vector_clock)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET
			title=excluded.title, content=excluded.content,
			mode=excluded.mode, color=excluded.color, bg_color=excluded.bg_color,
			pinned=excluded.pinned, width=excluded.width,
			height=excluded.height, pos_x=excluded.pos_x,
			pos_y=excluded.pos_y, updated_at=excluded.updated_at,
			deleted=excluded.deleted, synced=excluded.synced,
			version=excluded.version, vector_clock=excluded.vector_clock
	`, n.ID, n.UserID, n.Title, n.Content, n.Mode, n.Color, n.BgColor,
		pinned, n.Width, n.Height, n.PosX, n.PosY,
		n.CreatedAt, n.UpdatedAt, deleted, synced,
		n.Version, n.VectorClock,
	)
	return err
}

func GetUnsynced(userID string) ([]models.Note, error) {
	rows, err := DB.Query(
		`SELECT id,user_id,title,content,mode,color,pinned,
		        width,height,pos_x,pos_y,created_at,updated_at,
		        deleted,synced,version,vector_clock
		 FROM notes WHERE user_id = ? AND synced = 0`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []models.Note
	for rows.Next() {
		var n models.Note
		var pinned, deleted, synced int
		rows.Scan(
			&n.ID, &n.UserID, &n.Title, &n.Content, &n.Mode,
			&n.Color, &pinned, &n.Width, &n.Height,
			&n.PosX, &n.PosY, &n.CreatedAt, &n.UpdatedAt,
			&deleted, &synced, &n.Version, &n.VectorClock,
		)
		n.Pinned = pinned == 1
		n.Deleted = deleted == 1
		n.Synced = synced == 1
		notes = append(notes, n)
	}
	if notes == nil {
		notes = []models.Note{}
	}
	return notes, nil
}

func MarkSynced(id string) error {
	_, err := DB.Exec(`UPDATE notes SET synced = 1 WHERE id = ?`, id)
	return err
}

func GetLastSyncTime() int64 {
	var val string
	DB.QueryRow(`SELECT value FROM sync_meta WHERE key = 'last_sync'`).Scan(&val)
	var t int64
	fmt.Sscanf(val, "%d", &t)
	return t
}

func SetLastSyncTime(t int64) {
	DB.Exec(`
		INSERT INTO sync_meta (key,value) VALUES ('last_sync',?)
		ON CONFLICT(key) DO UPDATE SET value=excluded.value
	`, fmt.Sprintf("%d", t))
}

func GetNoteByID(id string) (*models.Note, error) {
	var n models.Note
	var pinned, deleted, synced int
	err := DB.QueryRow(`
		SELECT id,user_id,title,content,mode,color,bg_color,pinned,
			   width,height,pos_x,pos_y,created_at,updated_at,
			   deleted,synced,version,vector_clock
		FROM notes WHERE id = ?`, id,
	).Scan(
		&n.ID, &n.UserID, &n.Title, &n.Content, &n.Mode,
		&n.Color, &n.BgColor, &pinned, &n.Width, &n.Height,
		&n.PosX, &n.PosY, &n.CreatedAt, &n.UpdatedAt,
		&deleted, &synced, &n.Version, &n.VectorClock,
	)
	if err != nil {
		return nil, err
	}
	n.Pinned = pinned == 1
	n.Deleted = deleted == 1
	n.Synced = synced == 1
	return &n, nil
}