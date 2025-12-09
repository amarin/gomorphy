package storager

import (
	"io"

	"github.com/amarin/logging"
)

// Reader реализует последовательное чтение сегментов данных, заданных конфигурацией.
type Reader struct {
	config Config
	log    logging.Logger
}

// NewReader создаёт новый экземпляр Reader
func NewReader(name string, config Config) *Reader {
	return &Reader{
		config: config,
		log:    logging.NewNamedLogger(name),
	}
}

// ReadFrom читает данные сегментов из заданного io.Reader
func (reader Reader) ReadFrom(r io.Reader) (n int64, err error) {
	reader.log.Info("reading data")
	n = 0
	bytesTaken := int64(0)

	for _, segment := range reader.config.segments {
		reader.log.Debugf("reading segment %s", segment.name)

		bytesTaken, err = segment.Read(r)
		n += bytesTaken

		reader.log.Debugf("got %d bytes of segment %s data", bytesTaken, segment.name)
		if err != nil {
			return n, err
		}
	}

	reader.log.Info("loaded successfully")
	return n, err
}
