// Package common holds small helpers shared across gomorphy's source-data
// loaders (pkg/opencorpora, pkg/pymorphy): the on-disk data directory
// layout (GetDataPath/DomainDataPath/DomainFilePath) and a couple of
// constants used by HTTP-based downloaders.
package common

// HTTPHeaderLastModified is the standard HTTP header name used to detect
// whether a remote source archive has changed since the last download.
const HTTPHeaderLastModified = "Last-Modified"
