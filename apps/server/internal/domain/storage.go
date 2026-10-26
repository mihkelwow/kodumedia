package domain

import "time"

type StorageCategory struct {
	ID               string `json:"id"`
	SizeBytes        int64  `json:"sizeBytes"`
	LimitBytes       int64  `json:"limitBytes,omitempty"`
	ReclaimableBytes int64  `json:"reclaimableBytes,omitempty"`
	Clearable        bool   `json:"clearable,omitempty"`
	RetentionCount   int    `json:"retentionCount,omitempty"`
}

type StorageReport struct {
	DataDirectory        string            `json:"dataDirectory"`
	DataSizeBytes        int64             `json:"dataSizeBytes"`
	DiskFreeBytes        int64             `json:"diskFreeBytes"`
	DiskTotalBytes       int64             `json:"diskTotalBytes"`
	RequiredReserveBytes int64             `json:"requiredReserveBytes"`
	Categories           []StorageCategory `json:"categories"`
	GeneratedAt          time.Time         `json:"generatedAt"`
}

type StorageCleanupRequest struct {
	Action string `json:"action"`
}

type StorageCleanupResult struct {
	FreedBytes int64         `json:"freedBytes"`
	Report     StorageReport `json:"report"`
}
