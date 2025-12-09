package storager

// Config задаёт последовательность сегментов данных для чтения и записи из файла.
type Config struct {
	segments []Segment
}

// NewConfig создаёт новую конфигурацию чтения-записи сегментов.
func NewConfig(segments ...Segment) *Config {
	return &Config{segments: segments}
}
