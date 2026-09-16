package pymorphy

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// pypiRelease is one entry of PyPI JSON API's "urls" array.
type pypiRelease struct {
	URL         string `json:"url"`
	PackageType string `json:"packagetype"`
}

// pypiResponse is the subset of the PyPI JSON API response
// (https://pypi.org/pypi/<package>/json) this package needs.
type pypiResponse struct {
	Info struct {
		Version string `json:"version"`
	} `json:"info"`
	URLs []pypiRelease `json:"urls"`
}

// parsePyPIResponse extracts the release version and wheel download URL from
// a PyPI JSON API response body. Returns an error wrapping ErrPymorphy if the
// body doesn't parse or carries no bdist_wheel release.
func parsePyPIResponse(data []byte) (version string, wheelURL string, err error) {
	var payload pypiResponse
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", "", fmt.Errorf("%w: parse PyPI response: %w", ErrPymorphy, err)
	}

	for _, u := range payload.URLs {
		if u.PackageType == "bdist_wheel" {
			return payload.Info.Version, u.URL, nil
		}
	}

	return "", "", fmt.Errorf("%w: no wheel release found for %s", ErrPymorphy, PyPIPackageName)
}

// fetchLatestWheel queries jsonURL (PyPIJSONURL in production) for
// PyPIPackageName's latest release and returns its version and wheel URL.
func fetchLatestWheel(jsonURL string) (version string, wheelURL string, err error) {
	resp, err := http.Get(jsonURL) //nolint:gosec,noctx
	if err != nil {
		return "", "", fmt.Errorf("%w: fetch %s: %w", ErrPymorphy, jsonURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("%w: %s: unexpected status %d", ErrPymorphy, jsonURL, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("%w: read %s: %w", ErrPymorphy, jsonURL, err)
	}

	return parsePyPIResponse(data)
}
