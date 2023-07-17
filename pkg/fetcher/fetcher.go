package fetcher

import (
	"io"
	"net/http"
	"net/url"
	"time"
)

type Fetcher interface {
	FetchWebpageContent(url url.URL) (string, error)
}

type httpGetter interface {
	Get(url string) (resp *http.Response, err error)
}

type HTTPFetcher struct {
	httpClient httpGetter
}

type ExpBackoffRetryFetcher struct {
	innerFetcher        Fetcher
	numberOfRetries     int
	delayBetweenRetries time.Duration
}

func NewExpBackoffRetryFetcher(innerFetcher Fetcher, numberOfRetries int, delayBetweenRetries time.Duration) *ExpBackoffRetryFetcher {
	return &ExpBackoffRetryFetcher{innerFetcher: innerFetcher, numberOfRetries: numberOfRetries, delayBetweenRetries: delayBetweenRetries}
}

func NewHTTPFetcher(httpClient httpGetter) *HTTPFetcher {
	return &HTTPFetcher{httpClient: httpClient}
}

func (f *HTTPFetcher) FetchWebpageContent(url url.URL) (string, error) {
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

func (r *ExpBackoffRetryFetcher) FetchWebpageContent(url url.URL) (string, error) {
	var lastError error
	for i := 1; i <= r.numberOfRetries; i++ {
		webpageContent, err := r.innerFetcher.FetchWebpageContent(url)
		if err != nil {
			lastError = err
			time.Sleep((time.Duration(i) ^ 2) * r.delayBetweenRetries)
			continue
		}
		return webpageContent, nil
	}
	return "", lastError
}
