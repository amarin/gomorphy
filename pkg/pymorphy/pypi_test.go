package pymorphy

import (
	"errors"
	"testing"
)

const fixturePyPIResponse = `{
  "info": {"version": "2.4.417127.4579844"},
  "urls": [
    {
      "url": "https://files.pythonhosted.org/packages/aa/bb/pymorphy2_dicts_ru-2.4.417127.4579844.tar.gz",
      "filename": "pymorphy2-dicts-ru-2.4.417127.4579844.tar.gz",
      "packagetype": "sdist"
    },
    {
      "url": "https://files.pythonhosted.org/packages/cc/dd/pymorphy2_dicts_ru-2.4.417127.4579844-py2.py3-none-any.whl",
      "filename": "pymorphy2_dicts_ru-2.4.417127.4579844-py2.py3-none-any.whl",
      "packagetype": "bdist_wheel"
    }
  ]
}`

func TestParsePyPIResponse_OK(t *testing.T) {
	version, wheelURL, err := parsePyPIResponse([]byte(fixturePyPIResponse))
	if err != nil {
		t.Fatalf("parsePyPIResponse() error = %v", err)
	}
	if version != "2.4.417127.4579844" {
		t.Errorf("version = %q, want %q", version, "2.4.417127.4579844")
	}
	wantURL := "https://files.pythonhosted.org/packages/cc/dd/pymorphy2_dicts_ru-2.4.417127.4579844-py2.py3-none-any.whl"
	if wheelURL != wantURL {
		t.Errorf("wheelURL = %q, want %q", wheelURL, wantURL)
	}
}

func TestParsePyPIResponse_NoWheel(t *testing.T) {
	const noWheel = `{"info": {"version": "1.0"}, "urls": [{"url": "https://example.org/x.tar.gz", "packagetype": "sdist"}]}`

	_, _, err := parsePyPIResponse([]byte(noWheel))
	if !errors.Is(err, ErrPymorphy) {
		t.Fatalf("parsePyPIResponse() error = %v, want wrapping ErrPymorphy", err)
	}
}

func TestParsePyPIResponse_InvalidJSON(t *testing.T) {
	_, _, err := parsePyPIResponse([]byte("not json"))
	if !errors.Is(err, ErrPymorphy) {
		t.Fatalf("parsePyPIResponse() error = %v, want wrapping ErrPymorphy", err)
	}
}
