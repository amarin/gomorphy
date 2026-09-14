package opencorpora

// Progress callback — вызывается periodically во время сборки словаря.
// a: processed items, b: total items (0 если unknown).
// Для вывода в stderr: func(a, b int) { fmt.Fprintf(os.Stderr, "...") }
type Progress func(a, b int)
