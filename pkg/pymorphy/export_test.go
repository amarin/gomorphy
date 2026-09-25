package pymorphy

// SetURL points loader at a test server instead of the real source.
func SetURL(loader *Loader, url string) { loader.pypiURL = url }
