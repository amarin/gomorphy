package storage

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
func (writer Writer) WriteTo(r io.Writer) (n int64, err error) {
	writer.log.Info("writing data")
	n = 0
	bytesWritten := int64(0)

	for _, segment := range writer.config.segments {
		writer.log.Debugf("writing segment %s", segment.name)

		bytesWritten, err = segment.Write(r)
		n += bytesWritten

		writer.log.Debugf("put %d bytes of segment %s data", bytesWritten, segment.name)
		if err != nil {
			return n, err
		}
	}

	writer.log.Info("saved successfully")
	return n, err
}
