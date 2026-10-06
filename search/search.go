package search

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/blevesearch/bleve/v2"
	"github.com/dkotik/cuebook"
)

type Result struct {
	cuebook.ByteRange
	Path  string
	Entry cuebook.Entry
}

type Index interface {
	Include(filePath string, entry cuebook.Entry) error
	Query(string) ([]Result, error)
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

func (i *bleveIndex) Include(filePath string, entry cuebook.Entry) error {
	byteRange, err := cuebook.NewByteRange(entry.Value)
	if err != nil {
		return err
	}
	result := Result{
		ByteRange: byteRange,
		Path:      filePath,
		Entry:     entry,
	}

	// TODO: rewrite this as custom bleve.DocumentMapping to avoid having to serialize
	jsonBytes, err := entry.Value.MarshalJSON()
	if err != nil {
		return err
	}
	var jsonDoc interface{}
	if err = json.Unmarshal(jsonBytes, &jsonDoc); err != nil {
		return err
	}
	key := result.id()
	if err = i.Index.Index(key, jsonDoc); err != nil {
		return err
	}
	i.Entries.Store(key, result)
	return nil
}

func (i *bleveIndex) Query(searchQuery string) ([]Result, error) {
	found, err := i.Index.Search(bleve.NewSearchRequest(bleve.NewQueryStringQuery(searchQuery)))
	if err != nil {
		return nil, err
	}
	result := make([]Result, 0, found.Total)
	for _, hit := range found.Hits {
		stored, ok := i.Entries.Load(hit.ID)
		if !ok {
			continue
		}
		match, ok := stored.(Result)
		if !ok {
			continue
		}
		result = append(result, match)
	}
	return result, nil
}

func (r Result) id() string {
	return fmt.Sprintf("%q:%d:%d", r.Path, r.Head, r.Tail)
}
