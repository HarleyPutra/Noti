package db

import (
	"encoding/json"
	"noti/models"
	"os"
	"path/filepath"
	"sync"
)

var (
	NotesDir string
	fileMu   sync.RWMutex // Prevents reading a file at the exact millisecond it's being written
)

// InitStore creates the master Noti folders on the user's hard drive
func InitStore() error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	NotesDir = filepath.Join(configDir, "noti", "notes")
	
	// Create the directory if it doesn't exist
	return os.MkdirAll(NotesDir, 0755)
}

// UpsertNote saves the Note struct as a beautifully formatted JSON file
func UpsertNote(note models.Note) error {
	fileMu.Lock()
	defer fileMu.Unlock()

	filePath := filepath.Join(NotesDir, note.ID+".json")
	
	// MarshalIndent makes the JSON human-readable and clean
	data, err := json.MarshalIndent(note, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}

// GetNoteByID reads a single JSON file
func GetNoteByID(id string) (*models.Note, error) {
	fileMu.RLock()
	defer fileMu.RUnlock()

	filePath := filepath.Join(NotesDir, id+".json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var note models.Note
	if err := json.Unmarshal(data, &note); err != nil {
		return nil, err
	}

	return &note, nil
}

// GetNotes scans the folder and loads every JSON file into memory
func GetNotes(userID string) ([]models.Note, error) {
	fileMu.RLock()
	defer fileMu.RUnlock()

	var notes []models.Note

	files, err := os.ReadDir(NotesDir)
	if err != nil {
		return notes, err
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			data, err := os.ReadFile(filepath.Join(NotesDir, file.Name()))
			if err != nil {
				continue
			}

			var note models.Note
			if err := json.Unmarshal(data, &note); err == nil {
				// We still respect the "Deleted" flag for sync purposes!
				if !note.Deleted {
					notes = append(notes, note)
				}
			}
		}
	}

	return notes, nil
}

// GetUnsynced returns all notes that haven't been pushed to Google Drive yet
func GetUnsynced(userID string) ([]models.Note, error) {
	allNotes, err := GetNotes(userID)
	if err != nil {
		return nil, err
	}

	var unsynced []models.Note
	for _, n := range allNotes {
		if !n.Synced {
			unsynced = append(unsynced, n)
		}
	}
	return unsynced, nil
}

// MarkSynced tells the JSON file it has been successfully backed up
func MarkSynced(id string) error {
	note, err := GetNoteByID(id)
	if err != nil {
		return err
	}
	note.Synced = true
	return UpsertNote(*note) // Saves the updated state
}

type SyncMeta struct {
	LastSync int64 `json:"last_sync"`
}

// GetLastSyncTime reads the sync metadata file
func GetLastSyncTime() int64 {
	fileMu.RLock()
	defer fileMu.RUnlock()

	filePath := filepath.Join(NotesDir, "sync_meta.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return 0 // Default to 0 if it's the first time syncing
	}

	var meta SyncMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return 0
	}
	return meta.LastSync
}

// SetLastSyncTime updates the sync metadata file
func SetLastSyncTime(t int64) {
	fileMu.Lock()
	defer fileMu.Unlock()

	meta := SyncMeta{LastSync: t}
	data, err := json.MarshalIndent(meta, "", "  ")
	if err == nil {
		filePath := filepath.Join(NotesDir, "sync_meta.json")
		os.WriteFile(filePath, data, 0644)
	}
}