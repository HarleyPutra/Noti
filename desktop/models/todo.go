package models

type Todo struct {
	ID          string `json:"id"`
	UserID      string `json:"user_id"`
	Title       string `json:"title"`
	Notes       string `json:"notes"`
	Done        bool   `json:"done"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
	Deleted     bool   `json:"deleted"`
	Synced      bool   `json:"synced"`
	Version     int    `json:"version"`
	VectorClock string `json:"vector_clock"`
}