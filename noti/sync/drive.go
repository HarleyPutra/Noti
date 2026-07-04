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

// Push now accepts the exact list of mathematically merged notes to upload
func Push(userID string, notesToPush []models.Note) error {
	svc, err := getSvc()
	if err != nil {
		return err
	}

	// 1. Marshal the exact payload handed to us by the conflict resolver
	data, err := json.Marshal(notesToPush)
	if err != nil {
		return err
	}

	// 2. Upload to Google Drive
	fileID, _ := getFileID(svc)
	if fileID == "" {
		_, err = svc.Files.Create(&drive.File{
			Name:    driveFileName,
			Parents: []string{"appDataFolder"},
		}).Media(strings.NewReader(string(data))).Do()
	} else {
		_, err = svc.Files.Update(fileID, &drive.File{}).
			Media(strings.NewReader(string(data))).Do()
	}

	// If the network failed during upload, return the error so the queue tries again later
	if err != nil {
		return err
	}

	// 3. Record the exact time of successful sync
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