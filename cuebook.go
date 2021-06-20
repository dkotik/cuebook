package cuebook

// Book holds a list of maps.
type Book struct {
	sources []string
}

func NewBook(sources ...string) (*Book, error) {
	return &Book{sources: sources}, nil
}
