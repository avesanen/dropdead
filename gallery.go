package main

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"

	"github.com/boltdb/bolt"
)

var (
	GalleriesBucket = []byte("galleries")
)

type Gallery struct {
	Name  string
	Files []*File
}

type File struct {
	Name    string
	Url     string
	Type    string
	SubType string
}

func (d *Dropdead) LoadGallery(gid string) (*Gallery, error) {
	var g *Gallery
	err := d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(GalleriesBucket)
		if b == nil {
			return errors.New(fmt.Sprintf("Database bucket '%s' not found.", GalleriesBucket))
		}
		v := b.Get([]byte(gid))
		if len(v) == 0 {
			return errors.New(fmt.Sprintf("Gallery '%s' not found.", gid))
		}

		buf := bytes.NewBuffer([]byte(v))
		dec := gob.NewDecoder(buf)
		return dec.Decode(&g)
	})
	return g, err
}

func (d *Dropdead) SaveGallery(g *Gallery) error {
	err := d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(GalleriesBucket)
		buf := new(bytes.Buffer)
		enc := gob.NewEncoder(buf)
		if err := enc.Encode(g); err != nil {
			return err
		}
		return b.Put([]byte(g.Name), buf.Bytes())
	})
	return err
}
