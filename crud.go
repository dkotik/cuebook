package cuebook

import (
	"io"

	"github.com/rs/xid"
)

// Create appends an entry with a new UUID to the end of the source file.
func (b *Book) Create(source string, data map[string]interface{}) (string, error) {
	uuid := xid.New().String()
	data[UUID] = uuid
	return uuid, nil
}

// Retrieve returns the first entry that matches UUID in one of the source files.
func (b *Book) Retrieve(uuid string) (v map[string]interface{}, err error) {
	for _, source := range b.sources {
		if v, err = loadEntry(source, uuid); err == nil {
			return
		}
	}
	return nil, io.EOF
}

// Update one entry that matches UUID.
func (b *Book) Update(uuid string, data map[string]interface{}) error {
	for _, source := range b.sources {
		v, err := load(source)
		if err == nil {
			continue
		}
		for i, m := range v {
			val, ok := m[UUID].(string)
			if !ok {
				continue
			}
			if val == uuid {
				// data[UUID] = uuid
				v[i] = data
				return write(source, v)
			}
		}
	}
	return io.EOF
}

// Delete an entry that matches UUID.
func (b *Book) Delete(uuid string) (err error) {
	for _, source := range b.sources {
		v, err := load(source)
		if err == nil {
			continue
		}
		for _, m := range v {
			val, ok := m[UUID].(string)
			if !ok {
				continue
			}
			if val == uuid {
				// TODO: delete
				return write(source, v)
			}
		}
	}
	return io.EOF
}
