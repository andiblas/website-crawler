# website-crawler

## Architecture overview
![Basic architecture](assets/crawler-architecture.png)

The implementation of this crawler consists of three main pieces.

#### [Fetcher](pkg/fetcher)
The fetcher component is in charge of retrieving the contents of a specific webpage. And just that.
You can see two different fetcher implementations: HTTPFetcher and ExpBackoffRetryFetcher.

#### [Link Extractor](pkg/linkextractor)
This component deals with the extraction of the links from the retrieved webpage. 
In this case we implemented an HTML extractor that traverses all the webpage contents and gets all the links from a specific domain.

#### [Crawler](pkg/crawler)
The crawler itself is the one in charge of crawling a specific page using both the Fetcher and LinkExtractor.
The crawling it's done in a Breadth First fashion, starting from the provided URL, and then browsing all the links referenced in the parent link from the current depth. 
When all those links are crawled, the crawler jumps into the next depth level. You can control how many times the crawler will continue jumping into deeper levels with the
`--depth` argument.

## How to use

### Crawl
```shell
make URL=https://parserdigital.com
```

#### Arguments
- `URL` URL to crawl
- `DEPTH` Sets the crawling depth. The depth is delimited by each time the crawler continues crawling on new discovered pages. Must be greater than 0.
- `MAX_CONCURRENCY` Sets the maximum concurrent requests the crawler can do. Must be greater than 0.
- `TIMEOUT` Request timeout used to get webpages in milliseconds. Must be greater than 0.
- `RETRIES` The number of retries the crawler will try to fetch a page in case of errors. Must be 0 or greater than 0.

### Run tests
```shell
make tests
```

