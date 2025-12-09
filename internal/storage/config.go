package storage

// Config задаёт последовательность сегментов данных для чтения и записи из файла.
type Config struct {
	segments []DataSegment
}

// Define создаёт новую конфигурацию чтения-записи сегментов.
func Define(segments ...DataSegment) *Config {
	return &Config{segments: segments}
}
