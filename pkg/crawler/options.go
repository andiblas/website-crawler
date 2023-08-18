package crawler

type Option func(crawler *BreadthFirstCrawler)

func WithLinkFoundCallback(linkFound linkFoundCallback) Option {
	return func(crawler *BreadthFirstCrawler) {
		crawler.linkFound = linkFound
	}
}

func WithOnErrorCallback(onErrorCallback crawlingErrorCallback) Option {
	return func(crawler *BreadthFirstCrawler) {
		crawler.onError = onErrorCallback
	}
}
