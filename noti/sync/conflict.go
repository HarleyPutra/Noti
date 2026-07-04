package sync

import (
	"encoding/json"
	"noti/models"

	"github.com/google/uuid"
)

// MergeNotes handles the vector math and catches any forked conflicted copies
func MergeNotes(local, remote []models.Note) []models.Note {
	index := make(map[string]models.Note)
	var forkedNotes []models.Note // Array to catch duplicated subway collisions!

	for _, n := range local {
		index[n.ID] = n
	}
	
	for _, r := range remote {
		l, exists := index[r.ID]
		if !exists {
			index[r.ID] = r
		} else {
			merged, fork := mergeSingle(l, r)
			index[r.ID] = merged
			if fork != nil {
				forkedNotes = append(forkedNotes, *fork)
			}
		}
	}
	
	result := make([]models.Note, 0, len(index)+len(forkedNotes))
	for _, n := range index {
		result = append(result, n)
	}
	// Append our safe duplicated copies to the final save list!
	result = append(result, forkedNotes...)
	return result
}

// mergeSingle returns the primary note, and an optional *Note if a fork was required
func mergeSingle(local, remote models.Note) (models.Note, *models.Note) {
	lc := parseClock(local.VectorClock)
	rc := parseClock(remote.VectorClock)
	
	switch compareClocks(lc, rc) {
	case "local-newer":
		return local, nil
	case "remote-newer":
		return remote, nil
	default:
		// 🚨 CONCURRENT SUBWAY COLLISION!
		// We refuse to overwrite data. We keep the remote as the main note, 
		// and duplicate the local changes as a safe conflicted copy.

		// 1. The Primary (Remote wins the original ID, but clock fast-forwards)
		merged := remote
		merged.VectorClock = marshalClock(mergeClock(lc, rc))
		merged.Deleted = local.Deleted || remote.Deleted
		merged.Version = max(local.Version, remote.Version) + 1

		// 2. The Fork (Local is saved safely under a new ID)
		fork := local
		fork.ID = uuid.New().String()
		fork.Title = "[Conflicted Copy] " + fork.Title
		fork.VectorClock = `{"` + local.UserID + `":1}` // Reset clock for the new note
		fork.Synced = false
		fork.Version = 1

		return merged, &fork
	}
}

// ... compareClocks, mergeClock, parseClock, marshalClock, and max remain exactly the same!
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