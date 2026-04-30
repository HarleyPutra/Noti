package models

type Note struct {
	ID          string `json:"id"`
	UserID      string `json:"user_id"`
	Title       string `json:"title"`
	Content     string `json:"content"`
	Mode        string `json:"mode"`    // list, lined, squares, dots, browse
	Color       string `json:"color"`   // header color hex
	Pinned      bool   `json:"pinned"`  // always on top
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	PosX        int    `json:"pos_x"`
	PosY        int    `json:"pos_y"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
	Deleted     bool   `json:"deleted"`
	Synced      bool   `json:"synced"`
	Version     int    `json:"version"`
	VectorClock string `json:"vector_clock"`
}

type TimerState struct {
	NoteID    string `json:"note_id"`
	Minutes   int    `json:"minutes"`
	Seconds   int    `json:"seconds"`
	Running   bool   `json:"running"`
	Mode      string `json:"mode"` // pomodoro, countdown
	BreakTime bool   `json:"break_time"`
}