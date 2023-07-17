package fetcher

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

type mockHttpGetter struct {
	webpageContent string
	throwError     error
}

func (m mockHttpGetter) Get(_ string) (resp *http.Response, err error) {
	return &http.Response{
		Body: io.NopCloser(strings.NewReader(m.webpageContent)),
	}, m.throwError
}

func TestHTTPFetcher_FetchWebpageContent(t *testing.T) {
	t.Run("returns webpagecontent with provided getter", func(t *testing.T) {
		mockWebpageContent := "<body><p>Test</p></body>"
		httpFetcher := NewHTTPFetcher(mockHttpGetter{
			webpageContent: mockWebpageContent,
			throwError:     nil,
		})
		webpageContent, err := httpFetcher.FetchWebpageContent(url.URL{})
		if err != nil {
			t.Errorf("should not throw error at httpFetcher.FetchWebpageContent for mocked httpgetter. err: %v", err)
		}
		if !reflect.DeepEqual(webpageContent, mockWebpageContent) {
			t.Errorf("Extract() got = %v, want %v", webpageContent, mockWebpageContent)
		}
	})

	t.Run("returns an error from a failure in getter", func(t *testing.T) {
		httpFetcherError := NewHTTPFetcher(mockHttpGetter{
			webpageContent: "",
			throwError:     errors.New("mock error"),
		})
		_, err := httpFetcherError.FetchWebpageContent(url.URL{})
		if err == nil {
			t.Errorf("should throw error at httpFetcher.FetchWebpageContent for mocked httpgetter")
		}
	})
}
