package fetcher

import (
	"io"
	"net/http"
	"net/url"
)

type Fetcher interface {
	GetWebpageContent(url url.URL) (string, error)
}

type HTTPFetcher struct {
	httpClient httpGetter
}

type httpGetter interface {
	Get(url string) (resp *http.Response, err error)
}

func NewHTTPFetcher(httpClient httpGetter) *HTTPFetcher {
	return &HTTPFetcher{httpClient: httpClient}
}

func (f *HTTPFetcher) GetWebpageContent(url url.URL) (string, error) {
	res, err := f.httpClient.Get(url.String())
	if err != nil {
		return "", err
	}
	content, err := io.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		return "", err
	}

	return string(content), nil
}
