package git

// FileLinker maps a repository-relative file path to its application URL.
type FileLinker interface {
	GetFileLink(string) (string, error)
}
