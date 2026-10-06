package search

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/blevesearch/bleve/v2"
	"github.com/dkotik/cuebook"
)

type IndexKey struct {
	Index    int
	FilePath string
}

func (k IndexKey) String() string {
	return fmt.Sprintf("%d@%s", k.Index, k.FilePath)
}

type Index interface {
	Include(IndexKey, cuebook.Entry) error
	Query(string) ([]cuebook.Entry, error)
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

func (i *bleveIndex) Include(key IndexKey, entry cuebook.Entry) error {
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

func (i *bleveIndex) Query(searchQuery string) (result []cuebook.Entry, err error) {
	found, err := i.Index.Search(bleve.NewSearchRequest(bleve.NewQueryStringQuery(searchQuery)))
	if err != nil {
		return nil, err
	}
	result = make([]cuebook.Entry, 0, found.Total)
	for _, hit := range found.Hits {
		if entry, ok := i.Entries.Load(hit.ID); ok {
			result = append(result, entry.(cuebook.Entry))
		}
	}
	return result, nil
}
