package domain

import "time"

type MetadataTitleCandidate struct {
	Title string
	Year  int
}

type MetadataMatchQuery struct {
	MediaID           string
	Type              MediaType
	Title             string
	SeriesTitle       string
	Year              int
	Season            int
	Episode           int
	Language          string
	ExternalIDs       ExternalIDs
	ParentExternalIDs ExternalIDs
	Candidates        []MetadataTitleCandidate
}

type MetadataResult struct {
	ExternalIDs ExternalIDs
	Metadata    EditorialMetadata
	ReleaseDate *time.Time
	Year        int
	Title       string
}
