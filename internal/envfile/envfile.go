// Package envfile parses .env style files into variable names and values.
package envfile

// Var is a single declaration in an env file.
type Var struct {
	Key   string
	Value string
	Line  int
}

// File is a parsed env file.
type File struct {
	Path string
	Vars []Var
}

// Keys returns the declared variable names in file order.
func (f *File) Keys() []string {
	keys := make([]string, 0, len(f.Vars))
	for _, v := range f.Vars {
		keys = append(keys, v.Key)
	}
	return keys
}

// Parse reads the env file at path.
func Parse(path string) (*File, error) {
	// TODO: handle KEY=value, quoting, export prefixes, comments and blank lines.
	return &File{Path: path}, nil
}
