package crawler

type Option func(crawler *BreadthFirstCrawler)

// WithLinkFoundCallback is an option to set the callback function that
// will be executed when a new link is discovered during crawling.
//
// Parameters:
//   - linkFound: The linkFoundCallback function to set as the link discovery callback.
//
// Returns:
//   - An Option function that sets the provided linkFoundCallback to the BreadthFirstCrawler.
//
// Example usage:
//
//	linkCallback := func(link url.URL) {
//	    fmt.Println("Link discovered:", link.String())
//	}
//	crawler := NewBreadthFirstCrawler(fetcher, WithLinkFoundCallback(linkCallback))
func WithLinkFoundCallback(linkFound linkFoundCallback) Option {
	return func(crawler *BreadthFirstCrawler) {
		crawler.linkFound = linkFound
	}
}

// WithOnErrorCallback is an option to set the callback function that
// will be executed when an error occurs during crawling.
//
// Parameters:
//   - onErrorCallback: The crawlingErrorCallback function to set as the error callback.
//
// Returns:
//   - An Option function that sets the provided crawlingErrorCallback to the BreadthFirstCrawler.
//
// Example usage:
//
//	errorCallback := func(link url.URL, err error) {
//	    fmt.Println("Error occurred while crawling link:", link.String(), "Error:", err)
//	}
//	crawler := NewBreadthFirstCrawler(fetcher, WithOnErrorCallback(errorCallback))
func WithOnErrorCallback(onErrorCallback crawlingErrorCallback) Option {
	return func(crawler *BreadthFirstCrawler) {
		crawler.onError = onErrorCallback
	}
}
