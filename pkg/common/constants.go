// Package common holds small helpers shared across gomorphy's source-data
// loaders (pkg/opencorpora, pkg/pymorphy, pkg/unimorph): the on-disk data
// directory layout (GetDataPath/DomainDataPath/DomainFilePath),
// NewLoaderLogger (a silent logger when logging.Init was not called) and
// the HTTPHeaderLastModified constant.
package common

// HTTPHeaderLastModified is the standard HTTP header name used to detect
// whether a remote source archive has changed since the last download.
const HTTPHeaderLastModified = "Last-Modified"
