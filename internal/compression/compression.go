package compression

import (
	"bytes"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"io"
	"strings"
)

func Compress(codec kafka.CompressionCodec, src []byte) ([]byte, error) {
	b := new(bytes.Buffer)
	r := bytes.NewReader(src)
	w := codec.NewWriter(b)
	if _, err := io.Copy(w, r); err != nil {
		_ = w.Close()
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func Decompress(codec kafka.CompressionCodec, src []byte) ([]byte, error) {
	b := new(bytes.Buffer)
	r := codec.NewReader(bytes.NewReader(src))
	if _, err := io.Copy(b, r); err != nil {
		_ = r.Close()
		return nil, err
	}
	if err := r.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func RandomString(targetSize int) (string, error) {
	b := strings.Builder{}
	b.Grow(targetSize)
	for b.Len() < targetSize {
		data := uuid.New().String()
		if _, err := b.WriteString(data); err != nil {
			return "", err
		}
	}

	return b.String(), nil
}

