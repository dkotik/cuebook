package cuebook

import "io"

func (b *Book) Create(source string, data map[string]interface{}) error {
	return nil
}

func (b *Book) Retrieve(uuid string) (v map[string]interface{}, err error) {
	for _, source := range b.sources {
		if v, err = loadEntry(source, uuid); err == nil {
			return
		}
	}
	return nil, io.EOF
}

func (b *Book) Update(uuid string, data map[string]interface{}) error {
	return nil
}

func (b *Book) Delete(uuid string) error {
	return nil
}
