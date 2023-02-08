package data

import "time"

type Movie struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"-"` // Timestamp of when movie was first added to the database.
	Title     string    `json:"title"`
	Year      int32     `json:"year,omitempty"`    // Year of release.
	Runtime   int32     `json:"runtime,omitempty"` // Runtime in minutes.
	Genres    []string  `json:"genres,omitempty"`
	Version   int32     `json:"version"` // Starts at 1, increments with every update.
}
