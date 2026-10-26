package domain

import "time"

type LibraryKind string

const (
	LibraryKindMovies LibraryKind = "movies"
	LibraryKindSeries LibraryKind = "series"
	LibraryKindMusic  LibraryKind = "music"
	LibraryKindMixed  LibraryKind = "mixed"
)

func (k LibraryKind) IsValid() bool {
	switch k {
	case LibraryKindMovies, LibraryKindSeries, LibraryKindMusic, LibraryKindMixed:
		return true
	default:
		return false
	}
}

type LibrarySource struct {
	ID        string    `json:"id"`
	Path      string    `json:"path"`
	CreatedAt time.Time `json:"createdAt"`
}

type Library struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Kind      LibraryKind     `json:"kind"`
	System    bool            `json:"system"`
	Sources   []LibrarySource `json:"sources"`
	CreatedAt time.Time       `json:"createdAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
}
