package scanner

import "path/filepath"

// jsScanner matches process.env.X, process.env["X"] and import.meta.env.X.
type jsScanner struct{}

func (jsScanner) Name() string { return "js" }

func (jsScanner) Supports(path string) bool {
	switch filepath.Ext(path) {
	case ".js", ".jsx", ".mjs", ".cjs", ".ts", ".tsx":
		return true
	}
	return false
}

func (jsScanner) Scan(path string, src []byte) ([]Ref, error) {
	// TODO: match process.env access and import.meta.env access.
	return nil, nil
}
