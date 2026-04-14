package sync

import (
	"context"
	"desktop/auth"
	"desktop/db"
	"desktop/models"
	"encoding/json"
	"io"
	"strings"
	"time"
    "fmt"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

const driveFileName = "todos_sync.json"

func getSvc() (*drive.Service, error) {
	client := auth.GetClient()
	if client == nil {
		return nil, fmt.Errorf("not authenticated")
	}
	return drive.NewService(
		context.Background(),
		option.WithHTTPClient(client),
	)
}

func Pull() ([]models.Todo, error) {
	svc, err := getSvc()
	if err != nil {
		return nil, err
	}

	fileID, err := getFileID(svc)
	if err != nil || fileID == "" {
		return nil, err
	}

	res, err := svc.Files.Get(fileID).Download()
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var todos []models.Todo
	if err := json.Unmarshal(data, &todos); err != nil {
		return nil, err
	}
	return todos, nil
}

func Push(userID string) error {
	svc, err := getSvc()
	if err != nil {
		return err
	}

	// Get all todos (not just unsynced) so Drive always has full picture
	todos, err := db.GetTodos(userID)
	if err != nil {
		return err
	}
	// Also include soft-deleted ones for sync purposes
	unsynced, _ := db.GetUnsynced(userID)
	merged := MergeTodos(todos, unsynced)

	data, err := json.Marshal(merged)
	if err != nil {
		return err
	}

	fileID, _ := getFileID(svc)
	if fileID == "" {
		// Create new file in appDataFolder
		_, err = svc.Files.Create(&drive.File{
			Name:    driveFileName,
			Parents: []string{"appDataFolder"},
		}).Media(strings.NewReader(string(data))).Do()
	} else {
		// Update existing file
		_, err = svc.Files.Update(fileID, &drive.File{}).
			Media(strings.NewReader(string(data))).Do()
	}
	if err != nil {
		return err
	}

	// Mark all as synced
	for _, t := range unsynced {
		db.MarkSynced(t.ID)
	}
	db.SetLastSyncTime(time.Now().UnixMilli())
	return nil
}

func getFileID(svc *drive.Service) (string, error) {
	list, err := svc.Files.List().
		Spaces("appDataFolder").
		Q("name = '" + driveFileName + "'").
		Fields("files(id)").
		Do()
	if err != nil {
		return "", err
	}
	if len(list.Files) > 0 {
		return list.Files[0].Id, nil
	}
	return "", nil
}