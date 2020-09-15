package common

// DomainDataLoader requires implementation declares methods to deal with remote data and storage path.
type DomainDataLoader interface {
	GetDataPath() string
}

type Dictionary interface {
	Load(filePath string) error // Load dictionary from specified path
}
