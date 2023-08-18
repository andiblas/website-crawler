package crawler

import (
	"io"
	"net/url"
	"testing"
)

type MockFetcher struct{}

func (f *MockFetcher) FetchWebpageContent(_ url.URL) (io.ReadCloser, error) {
	return nil, nil
}

func TestNewBreadthFirstCrawler(t *testing.T) {
	mockFetcher := &MockFetcher{}

	linkFoundMock := func(link url.URL) {}
	onErrorMock := func(link url.URL, err error) {}

	crawler := NewBreadthFirstCrawler(mockFetcher, WithLinkFoundCallback(linkFoundMock), WithOnErrorCallback(onErrorMock))

	if crawler.fetcher != mockFetcher {
		t.Errorf("Expected fetcher to be set to mockFetcher")
	}

	if crawler.linkFound == nil {
		t.Errorf("Expected linkFound callback to be set")
	}

	if crawler.onError == nil {
		t.Errorf("Expected onError callback to be set")
	}
}
