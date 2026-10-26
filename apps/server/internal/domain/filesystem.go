package domain

type FileSystemEntry struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	Directory bool   `json:"directory"`
	SizeBytes int64  `json:"sizeBytes,omitempty"`
}

type DirectoryListing struct {
	CurrentPath string            `json:"currentPath,omitempty"`
	ParentPath  string            `json:"parentPath,omitempty"`
	Roots       []FileSystemEntry `json:"roots"`
	Entries     []FileSystemEntry `json:"entries"`
	Truncated   bool              `json:"truncated,omitempty"`
}

type FilePickerMode string

const (
	FilePickerModeDirectory FilePickerMode = "directory"
	FilePickerModeFile      FilePickerMode = "file"
)

type FilePickerRequest struct {
	Mode        FilePickerMode `json:"mode"`
	Title       string         `json:"title"`
	InitialPath string         `json:"initialPath,omitempty"`
	Extensions  []string       `json:"extensions,omitempty"`
}

type FilePickerResult struct {
	Path string `json:"path"`
}
