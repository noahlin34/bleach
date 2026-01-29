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
	Report     []string
}

type Summary struct {
	Total      int
	Processed  int
	Errors     int
	Leaks      int
	BytesSaved int64
}

type ScanReport struct {
	Path       string
	Categories []string
}

type ProgressUpdate struct {
	TotalDelta      int
	ProcessedDelta  int
	ErrorDelta      int
	LeakDelta       int
	BytesSavedDelta int64
}
