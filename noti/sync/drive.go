package sync

import (
	"context"
	"encoding/json"
	"io"
	"noti/auth"
	"noti/db"
	"noti/models"
	"strings"
	"time"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"fmt"
)

const driveFileName = "noti_sync.json"

func getSvc() (*drive.Service, error) {
	client := auth.GetClient()
	if client == nil {
		return nil, fmt.Errorf("not authenticated")
	}
	return drive.NewService(context.Background(), option.WithHTTPClient(client))
}

func Pull() ([]models.Note, error) {
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
	data, _ := io.ReadAll(res.Body)
	var notes []models.Note
	json.Unmarshal(data, &notes)
	return notes, nil
}

func Push(userID string) error {
	svc, err := getSvc()
	if err != nil {
		return err
	}
	notes, err := db.GetNotes(userID)
	if err != nil {
		return err
	}
	data, _ := json.Marshal(notes)
	fileID, _ := getFileID(svc)
	if fileID == "" {
		svc.Files.Create(&drive.File{
			Name:    driveFileName,
			Parents: []string{"appDataFolder"},
		}).Media(strings.NewReader(string(data))).Do()
	} else {
		svc.Files.Update(fileID, &drive.File{}).
			Media(strings.NewReader(string(data))).Do()
	}
	unsynced, _ := db.GetUnsynced(userID)
	for _, n := range unsynced {
		db.MarkSynced(n.ID)
	}
	db.SetLastSyncTime(time.Now().UnixMilli())
	return nil
}

func getFileID(svc *drive.Service) (string, error) {
	list, err := svc.Files.List().
		Spaces("appDataFolder").
		Q("name = '" + driveFileName + "'").
		Fields("files(id)").Do()
	if err != nil {
		return "", err
	}
	if len(list.Files) > 0 {
		return list.Files[0].Id, nil
	}
	return "", nil
}