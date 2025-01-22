package cuebook

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/blevesearch/bleve/v2"
)

type IndexKey struct {
	Index    int
	FilePath string
}

func (k IndexKey) String() string {
	return fmt.Sprintf("%d@%s", k.Index, k.FilePath)
}

type Index interface {
	Include(IndexKey, Entry) error
	Query(string) ([]Entry, error)
}

func NewBleveIndex() Index {
	mapping := bleve.NewIndexMapping()
	index, err := bleve.NewMemOnly(mapping)
	if err != nil {
		panic(err)
	}
	return &bleveIndex{
		Index:   index,
		Entries: &sync.Map{},
	}
}

type bleveIndex struct {
	Index   bleve.Index
	Entries *sync.Map
}

func (i *bleveIndex) Include(key IndexKey, entry Entry) error {
	// TODO: rewrite this as custom bleve.DocumentMapping to avoid having to serialize
	jsonBytes, err := entry.Value.MarshalJSON()
	if err != nil {
		return err
	}
	var jsonDoc interface{}
	if err = json.Unmarshal(jsonBytes, &jsonDoc); err != nil {
		return err
	}
	k := key.String()
	i.Entries.Store(k, entry)
	return i.Index.Index(k, jsonDoc)
}

func (i *bleveIndex) Query(searchQuery string) (result []Entry, err error) {
	found, err := i.Index.Search(bleve.NewSearchRequest(bleve.NewQueryStringQuery(searchQuery)))
	if err != nil {
		return nil, err
	}
	result = make([]Entry, 0, found.Total)
	for _, hit := range found.Hits {
		if entry, ok := i.Entries.Load(hit.ID); ok {
			result = append(result, entry.(Entry))
		}
	}
	return result, nil
}
