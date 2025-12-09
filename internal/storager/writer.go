package storager

import (
	"io"

	"github.com/amarin/logging"
)

// Writer реализует последовательную запись сегментов данных, заданных конфигурацией.
type Writer struct {
	config Config
	log    logging.Logger
}

// NewWriter создаёт новый экземпляр Writer
func NewWriter(name string, config Config) *Writer {
	return &Writer{
		config: config,
		log:    logging.NewNamedLogger(name),
	}
}

// WriteTo записывает данные сегментов в заданный io.Writer
func (reader Reader) WriteTo(r io.Writer) (n int64, err error) {
	reader.log.Info("writing data")
	n = 0
	bytesWritten := int64(0)

	for _, segment := range reader.config.segments {
		reader.log.Debugf("writing segment %s", segment.name)

		bytesWritten, err = segment.Write(r)
		n += bytesWritten

		reader.log.Debugf("put %d bytes of segment %s data", bytesWritten, segment.name)
		if err != nil {
			return n, err
		}
	}

	reader.log.Info("saved successfully")
	return n, err
}
