package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"strconv"

	_ "modernc.org/sqlite"
	"noti/models"
)

var DB *sql.DB

// InitStore creates the database file and configures it for high concurrency
func InitStore() error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	dbDir := filepath.Join(configDir, "noti", "db")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return err
	}

	dbPath := filepath.Join(dbDir, "noti.db")
	
	// Open the database
	database, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}
	DB = database

	// CRITICAL DEADLOCK PREVENTION:
	// WAL mode allows concurrent reads and writes. Busy timeout prevents lock panics.
	DB.Exec(`PRAGMA journal_mode=WAL;`)
	DB.Exec(`PRAGMA busy_timeout=5000;`)
	DB.Exec(`PRAGMA synchronous=NORMAL;`)

	// Create Notes Table
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS notes (
			id TEXT PRIMARY KEY,
			user_id TEXT,
			title TEXT,
			content TEXT,
			mode TEXT,
			color TEXT,
			bg_color TEXT,
			pinned INTEGER,
			pos_x INTEGER,
			pos_y INTEGER,
			width INTEGER,
			height INTEGER,
			created_at INTEGER,
			updated_at INTEGER,
			deleted INTEGER,
			synced INTEGER,
			version INTEGER,
			vector_clock TEXT,
			timer_deadline INTEGER
		);
	`)
	if err != nil {
		return err
	}

	// Create the Mutation Queue Table
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS sync_queue (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			note_id TEXT,
			action TEXT,
			queued_at INTEGER
		);
	`)
	if err != nil {
		return err
	}

	// Create a simple Key-Value table for app metadata (like sync times)
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS meta (
			key TEXT PRIMARY KEY,
			value TEXT
		);
	`)
	return err
}

// UpsertNote saves the note AND queues it for sync in a single atomic transaction
func UpsertNote(note models.Note) error {
	// 1. Start a Transaction
	tx, err := DB.Begin()
	if err != nil {
		return err
	}

	// 2. Write the Note
	query := `
		INSERT INTO notes (
			id, user_id, title, content, mode, color, bg_color, pinned, 
			pos_x, pos_y, width, height, created_at, updated_at, 
			deleted, synced, version, vector_clock, timer_deadline
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			user_id=excluded.user_id, title=excluded.title, content=excluded.content,
			mode=excluded.mode, color=excluded.color, bg_color=excluded.bg_color,
			pinned=excluded.pinned, pos_x=excluded.pos_x, pos_y=excluded.pos_y,
			width=excluded.width, height=excluded.height, created_at=excluded.created_at,
			updated_at=excluded.updated_at, deleted=excluded.deleted, 
			synced=excluded.synced, version=excluded.version, 
			vector_clock=excluded.vector_clock, timer_deadline=excluded.timer_deadline;
	`
	_, err = tx.Exec(query,
		note.ID, note.UserID, note.Title, note.Content, note.Mode, note.Color, note.BgColor,
		boolToInt(note.Pinned), note.PosX, note.PosY, note.Width, note.Height,
		note.CreatedAt, note.UpdatedAt, boolToInt(note.Deleted), 0, // Force synced to 0 on edit
		note.Version, note.VectorClock, note.TimerDeadline,
	)
	if err != nil {
		tx.Rollback()
		return err
	}

	// 3. Queue the Event
	action := "UPDATE"
	if note.Deleted {
		action = "DELETE"
	}
	
	_, err = tx.Exec(`INSERT INTO sync_queue (note_id, action, queued_at) VALUES (?, ?, ?)`, 
		note.ID, action, note.UpdatedAt)
	if err != nil {
		tx.Rollback()
		return err
	}

	// 4. Commit everything safely!
	return tx.Commit()
}

// GetNoteByID fetches a single note
func GetNoteByID(id string) (*models.Note, error) {
	row := DB.QueryRow(`SELECT * FROM notes WHERE id = ?`, id)
	return scanNote(row)
}

// GetNotes fetches all active notes for a specific user
func GetNotes(userID string) ([]models.Note, error) {
	rows, err := DB.Query(`SELECT * FROM notes WHERE user_id = ? AND deleted = 0`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNotes(rows)
}

// GetUnsynced fetches all notes that need to be pushed to Google Drive
func GetUnsynced(userID string) ([]models.Note, error) {
	rows, err := DB.Query(`SELECT * FROM notes WHERE user_id = ? AND synced = 0`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNotes(rows)
}

// MarkSynced flips the sync flag to true
func MarkSynced(id string) error {
	_, err := DB.Exec(`UPDATE notes SET synced = 1 WHERE id = ?`, id)
	return err
}

// PermanentDelete physically wipes the row from the database (Hard Purge)
func PermanentDelete(id string) error {
	_, err := DB.Exec(`DELETE FROM notes WHERE id = ?`, id)
	return err
}

// --- Metadata Functions (Replaces sync_meta.json) ---

func GetLastSyncTime() int64 {
	var val string
	err := DB.QueryRow(`SELECT value FROM meta WHERE key = 'last_sync'`).Scan(&val)
	if err != nil {
		return 0
	}
	parsed, _ := strconv.ParseInt(val, 10, 64)
	return parsed
}

func SetLastSyncTime(t int64) {
	val := strconv.FormatInt(t, 10)
	DB.Exec(`
		INSERT INTO meta (key, value) VALUES ('last_sync', ?)
		ON CONFLICT(key) DO UPDATE SET value=excluded.value;
	`, val)
}

// --- Helpers ---

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func intToBool(i int) bool {
	return i == 1
}

// scanNote maps a single database row into the Note struct
func scanNote(scanner interface{ Scan(dest ...any) error }) (*models.Note, error) {
	var n models.Note
	var pinned, deleted, synced int

	err := scanner.Scan(
		&n.ID, &n.UserID, &n.Title, &n.Content, &n.Mode, &n.Color, &n.BgColor,
		&pinned, &n.PosX, &n.PosY, &n.Width, &n.Height,
		&n.CreatedAt, &n.UpdatedAt, &deleted, &synced,
		&n.Version, &n.VectorClock, &n.TimerDeadline,
	)
	if err != nil {
		return nil, err
	}

	n.Pinned = intToBool(pinned)
	n.Deleted = intToBool(deleted)
	n.Synced = intToBool(synced)
	return &n, nil
}

// scanNotes loops through multiple rows and maps them
func scanNotes(rows *sql.Rows) ([]models.Note, error) {
	var notes []models.Note
	for rows.Next() {
		note, err := scanNote(rows)
		if err == nil {
			notes = append(notes, *note)
		}
	}
	return notes, nil
}

type SyncEvent struct {
	ID       int
	NoteID   string
	Action   string
	QueuedAt int64
}

// GetSyncQueue fetches pending events, ordered by oldest first
func GetSyncQueue() ([]SyncEvent, error) {
	rows, err := DB.Query(`SELECT id, note_id, action, queued_at FROM sync_queue ORDER BY queued_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []SyncEvent
	for rows.Next() {
		var e SyncEvent
		if err := rows.Scan(&e.ID, &e.NoteID, &e.Action, &e.QueuedAt); err == nil {
			events = append(events, e)
		}
	}
	return events, nil
}

// ClearFromQueue removes an event once Google Drive has successfully received it
func ClearFromQueue(eventID int) error {
	_, err := DB.Exec(`DELETE FROM sync_queue WHERE id = ?`, eventID)
	return err
}