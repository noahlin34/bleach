package processor

type Mode int

const (
	ModeScan Mode = iota
	ModeClean
)

type Options struct {
	Mode        Mode
	InPlace     bool
	OutputDir   string
	PreserveICC bool
}

type Job struct {
	Path    string
	RelPath string
	Display string
}

type Result struct {
	Path       string
	RelPath    string
	Display    string
	Supported  bool
	Err        error
	Leaks      int
	BytesSaved int64
	Report     []ScanDetail
}

type Summary struct {
	Total      int
	Processed  int
	Errors     int
	Leaks      int
	BytesSaved int64
}

type ScanReport struct {
	Path    string
	Details []ScanDetail
}

type ScanDetail struct {
	Category string
	Values   []string
}

type ProgressUpdate struct {
	TotalDelta      int
	ProcessedDelta  int
	ErrorDelta      int
	LeakDelta       int
	BytesSavedDelta int64
}
