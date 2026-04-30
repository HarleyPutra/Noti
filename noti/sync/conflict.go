package sync

import (
	"encoding/json"
	"noti/models"
)

func MergeNotes(local, remote []models.Note) []models.Note {
	index := make(map[string]models.Note)
	for _, n := range local {
		index[n.ID] = n
	}
	for _, r := range remote {
		l, exists := index[r.ID]
		if !exists {
			index[r.ID] = r
		} else {
			index[r.ID] = mergeSingle(l, r)
		}
	}
	result := make([]models.Note, 0, len(index))
	for _, n := range index {
		result = append(result, n)
	}
	return result
}

func mergeSingle(local, remote models.Note) models.Note {
	lc := parseClock(local.VectorClock)
	rc := parseClock(remote.VectorClock)
	switch compareClocks(lc, rc) {
	case "local-newer":
		return local
	case "remote-newer":
		return remote
	default:
		merged := local
		if remote.UpdatedAt > local.UpdatedAt {
			merged.Title   = remote.Title
			merged.Content = remote.Content
			merged.Mode    = remote.Mode
			merged.Color   = remote.Color
		}
		merged.Deleted     = local.Deleted || remote.Deleted
		if local.UpdatedAt > remote.UpdatedAt {
			merged.UpdatedAt = local.UpdatedAt
		} else {
			merged.UpdatedAt = remote.UpdatedAt
		}
		merged.Version     = max(local.Version, remote.Version) + 1
		merged.VectorClock = marshalClock(mergeClock(lc, rc))
		merged.Synced      = false
		return merged
	}
}

func compareClocks(a, b map[string]int) string {
	aGtB, bGtA := false, false
	for k := range a {
		if a[k] > b[k] { aGtB = true }
		if b[k] > a[k] { bGtA = true }
	}
	for k := range b {
		if b[k] > a[k] { bGtA = true }
		if a[k] > b[k] { aGtB = true }
	}
	if aGtB && !bGtA { return "local-newer" }
	if bGtA && !aGtB { return "remote-newer" }
	return "concurrent"
}

func mergeClock(a, b map[string]int) map[string]int {
	result := map[string]int{}
	for k, v := range a { result[k] = v }
	for k, v := range b {
		if v > result[k] { result[k] = v }
	}
	return result
}

func parseClock(s string) map[string]int {
	m := map[string]int{}
	json.Unmarshal([]byte(s), &m)
	return m
}

func marshalClock(m map[string]int) string {
	b, _ := json.Marshal(m)
	return string(b)
}

func max(a, b int) int {
	if a > b { return a }
	return b
}