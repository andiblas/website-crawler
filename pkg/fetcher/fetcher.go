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

// FetchWebpageContent fetches the content of a webpage specified by the given URL using an HTTP GET request.
// It uses the HTTP client provided in the HTTPFetcher and returns the content as a string.
// The method returns an error if the HTTP request fails or if there is an error reading the response body.
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

// FetchWebpageContent fetches the content of a webpage specified by the given URL using an exponential backoff retry strategy.
// It uses the innerFetcher to perform the actual fetch operation and retries fetching up to the specified number of times.
// The method returns the webpage content as a string and nil for the error if the fetch is successful.
// If the fetch encounters errors on all retries, the last encountered error is returned.
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
