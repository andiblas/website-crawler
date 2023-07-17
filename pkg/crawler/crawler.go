package crawler

import "net/url"

type Crawler interface {
	Crawl(url url.URL) ([]string, error)
}
