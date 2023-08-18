package crawler

import (
	"context"
	"errors"
	"net/url"
)

// InvalidDepth indicates that the provided depth for the crawl operation is invalid.
// The depth value must be greater than 0.
var InvalidDepth = errors.New("invalid depth. must be greater than 0")

// InvalidMaxConcurrency indicates that the provided maximum concurrency value for
// the crawl operation is invalid. The maxConcurrency value must be greater than 0
// to allow concurrent crawling of multiple pages.
var InvalidMaxConcurrency = errors.New("invalid maximum concurrency. must be greater than 0")

type Crawler interface {
	Crawl(ctx context.Context, urlToCrawl url.URL, depth, maxConcurrency int) ([]string, error)
}
