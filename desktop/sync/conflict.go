package sync

import (
	"desktop/models"
	"encoding/json"
)

func MergeTodos(local, remote []models.Todo) []models.Todo {
	index := make(map[string]models.Todo)

	for _, t := range local {
		index[t.ID] = t
	}

	for _, r := range remote {
		l, exists := index[r.ID]
		if !exists {
			index[r.ID] = r
		} else {
			index[r.ID] = mergeSingle(l, r)
		}
	}

	result := make([]models.Todo, 0, len(index))
	for _, t := range index {
		result = append(result, t)
	}
	return result
}

func mergeSingle(local, remote models.Todo) models.Todo {
	lc := parseClock(local.VectorClock)
	rc := parseClock(remote.VectorClock)
	rel := compareClocks(lc, rc)

	switch rel {
	case "local-newer":
		return local
	case "remote-newer":
		return remote
	default:
		// Concurrent — merge field by field
		merged := local
		if remote.UpdatedAt > local.UpdatedAt {
			merged.Title = remote.Title
			merged.Notes = remote.Notes
		}
		merged.Done    = local.Done || remote.Done
		merged.Deleted = local.Deleted || remote.Deleted
		if remote.UpdatedAt > local.UpdatedAt {
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
	result := make(map[string]int)
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