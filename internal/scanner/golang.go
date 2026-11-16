package scanner

import "path/filepath"

// goScanner matches os.Getenv("X") and os.LookupEnv("X").
type goScanner struct{}

func (goScanner) Name() string { return "go" }

func (goScanner) Supports(path string) bool {
	return filepath.Ext(path) == ".go"
}

func (goScanner) Scan(path string, src []byte) ([]Ref, error) {
	// TODO: match os.Getenv and os.LookupEnv calls with literal arguments.
	return nil, nil
}
