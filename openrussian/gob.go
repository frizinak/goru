package openrussian

import (
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"time"
)

func init() {
	gob.Register(Words{})
}

func EncodeGOB(w io.Writer, words Words) error {
	return gob.NewEncoder(w).Encode(words)
}

func DecodeGOB(r io.Reader) (Words, error) {
	w := Words{}
	return w, gob.NewDecoder(r).Decode(&w)
}

func StoreGOB(file string, words Words) error {
	tmp := fmt.Sprintf("%s.%d.tmp", file, time.Now().UnixNano())
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}

	if err := EncodeGOB(f, words); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}

	f.Close()
	return os.Rename(tmp, file)
}

func LoadGOB(file string) (Words, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}

	w, err := DecodeGOB(f)
	f.Close()
	return w, err
}
