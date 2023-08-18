package crawler

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/andiblas/website-crawler/pkg/fetcher"
)

type errorCallbackArgs struct {
	link url.URL
	err  error
}

type mockFetcher struct {
	webpageWithLinks map[string]string
	throwError       error
}

func newMockFetcher(throwError error) *mockFetcher {
	webpageWithLinks := map[string]string{
		"https://test.com":          `<a href="https://test.com"/><a href="https://test.com/contact"/><a href="https://test.com/about-us"/>`,
		"https://test.com/contact":  `<a href="https://test.com"/><a href="https://test.com/depth3"/>`,
		"https://test.com/about-us": `<a href="https://test.com"/><a href="https://test.com/contact"/><a href="https://test.com/about-us"/>`,
		"https://test.com/depth3":   `<a href="https://test.com"/><a href="https://test.com/depth4"/>`,
	}
	return &mockFetcher{webpageWithLinks: webpageWithLinks, throwError: throwError}
}

func (m mockFetcher) FetchWebpageContent(urlToCrawl url.URL) (io.ReadCloser, error) {
	if webpageHtml, ok := m.webpageWithLinks[urlToCrawl.String()]; ok {
		return io.NopCloser(strings.NewReader(webpageHtml)), m.throwError
	}
	return io.NopCloser(strings.NewReader("")), m.throwError
}

func TestBreadthFirstCrawler_Crawl(t *testing.T) {
	startUrl := "https://test.com"
	canceledCtx, cancelFunc := context.WithCancel(context.Background())
	cancelFunc()
	testUrl, _ := url.Parse(startUrl)
	linkFoundCh := make(chan url.URL)
	errorCallbackCh := make(chan errorCallbackArgs)
	type fields struct {
		fetcher fetcher.Fetcher
	}
	type args struct {
		ctx            context.Context
		urlToCrawl     url.URL
		depth          int
		maxConcurrency int
		linkFound      linkFoundCallback
		errorCallback  crawlingErrorCallback
	}
	tests := []struct {
		name          string
		fields        fields
		args          args
		want          map[string]bool
		wantLinkFound map[string]bool
		wantErr       bool
	}{
		{
			name:   "crawls with only one depth step and returns only one link (the provided url)",
			fields: fields{newMockFetcher(nil)},
			args: args{
				ctx:            context.Background(),
				urlToCrawl:     *testUrl,
				depth:          1,
				maxConcurrency: 1,
				errorCallback:  nil,
				linkFound:      nil,
			},
			want: map[string]bool{
				"https://test.com":          true,
				"https://test.com/contact":  true,
				"https://test.com/about-us": true,
			},
			wantLinkFound: nil,
			wantErr:       false,
		},
		{
			name:   "crawls up two depth levels and returns all pages at that depth level without repeating links and ignoring deeper links",
			fields: fields{newMockFetcher(nil)},
			args: args{
				ctx:            context.Background(),
				urlToCrawl:     *testUrl,
				depth:          2,
				maxConcurrency: 1,
				errorCallback:  nil,
				linkFound:      nil,
			},
			want: map[string]bool{
				"https://test.com":          true,
				"https://test.com/contact":  true,
				"https://test.com/about-us": true,
				"https://test.com/depth3":   true,
			},
			wantLinkFound: nil,
			wantErr:       false,
		},
		{
			name:   "crawling way too deep should get all links",
			fields: fields{newMockFetcher(nil)},
			args: args{
				ctx:            context.Background(),
				urlToCrawl:     *testUrl,
				depth:          100,
				maxConcurrency: 1,
				errorCallback:  nil,
				linkFound:      nil,
			},
			want: map[string]bool{
				"https://test.com":          true,
				"https://test.com/contact":  true,
				"https://test.com/about-us": true,
				"https://test.com/depth3":   true,
				"https://test.com/depth4":   true,
			},
			wantLinkFound: nil,
			wantErr:       false,
		},
		{
			name:   "invalid depth",
			fields: fields{newMockFetcher(nil)},
			args: args{
				ctx:            context.Background(),
				urlToCrawl:     *testUrl,
				depth:          -1,
				maxConcurrency: 1,
				errorCallback:  nil,
				linkFound:      nil,
			},
			want:          map[string]bool{},
			wantLinkFound: nil,
			wantErr:       true,
		},
		{
			name:   "invalid max concurrency",
			fields: fields{newMockFetcher(nil)},
			args: args{
				ctx:            context.Background(),
				urlToCrawl:     *testUrl,
				depth:          1,
				maxConcurrency: -1,
				errorCallback:  nil,
				linkFound:      nil,
			},
			want:          map[string]bool{},
			wantLinkFound: nil,
			wantErr:       true,
		},
		{
			name:   "crawl with canceled context gets interrupted and should not return no links",
			fields: fields{newMockFetcher(nil)},
			args: args{
				ctx:            canceledCtx,
				urlToCrawl:     *testUrl,
				depth:          1,
				maxConcurrency: 1,
				errorCallback:  nil,
				linkFound:      nil,
			},
			want:          map[string]bool{},
			wantLinkFound: nil,
			wantErr:       false,
		},
		{
			name:   "crawl calls linkFound callback for each link found",
			fields: fields{newMockFetcher(nil)},
			args: args{
				ctx:            context.Background(),
				urlToCrawl:     *testUrl,
				depth:          2,
				maxConcurrency: 1,
				errorCallback:  nil,
				linkFound: func(link url.URL) {
					fmt.Println("executing link found callback for", link.String())
					linkFoundCh <- link
				},
			},
			want: map[string]bool{
				"https://test.com":          true,
				"https://test.com/contact":  true,
				"https://test.com/about-us": true,
				"https://test.com/depth3":   true,
			},
			wantLinkFound: map[string]bool{
				"https://test.com/contact":  true,
				"https://test.com/about-us": true,
				"https://test.com/depth3":   true,
			},
			wantErr: false,
		},
		{
			name:   "crawl safely calls a panicking linkFound callback",
			fields: fields{newMockFetcher(nil)},
			args: args{
				ctx:            context.Background(),
				urlToCrawl:     *testUrl,
				depth:          2,
				maxConcurrency: 1,
				errorCallback:  nil,
				linkFound: func(link url.URL) {
					linkFoundCh <- link
					panic("")
				},
			},
			want: map[string]bool{
				"https://test.com":          true,
				"https://test.com/contact":  true,
				"https://test.com/about-us": true,
				"https://test.com/depth3":   true,
			},
			wantLinkFound: map[string]bool{
				"https://test.com/contact":  true,
				"https://test.com/about-us": true,
				"https://test.com/depth3":   true,
			},
			wantErr: false,
		},
		{
			name:   "crawl calls error callback when an error occurs while fetching",
			fields: fields{newMockFetcher(errors.New("error fetching"))},
			args: args{
				ctx:            context.Background(),
				urlToCrawl:     *testUrl,
				depth:          1,
				maxConcurrency: 1,
				errorCallback: func(link url.URL, err error) {
					errorCallbackCh <- errorCallbackArgs{link: link, err: err}
				},
				linkFound: nil,
			},
			want: map[string]bool{
				"https://test.com": true,
			},
			wantErr: false,
		},
		{
			name:   "crawl safely calls a panicking error callback",
			fields: fields{newMockFetcher(errors.New("error fetching"))},
			args: args{
				ctx:            context.Background(),
				urlToCrawl:     *testUrl,
				depth:          1,
				maxConcurrency: 1,
				errorCallback: func(link url.URL, err error) {
					errorCallbackCh <- errorCallbackArgs{link: link, err: err}
					panic("")
				},
				linkFound: nil,
			},
			want: map[string]bool{
				"https://test.com": true,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewBreadthFirstCrawler(tt.fields.fetcher, WithLinkFoundCallback(tt.args.linkFound), WithOnErrorCallback(tt.args.errorCallback))
			got, err := a.Crawl(tt.args.ctx, tt.args.urlToCrawl, tt.args.depth, tt.args.maxConcurrency)
			if (err != nil) != tt.wantErr {
				t.Errorf("Crawl() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != len(tt.want) {
				t.Errorf("Crawl() links len got %v want len %v\ngot\t\t%v\nwant\t%v", len(got), len(tt.want), got, tt.want)
			}
			for _, link := range got {
				if _, ok := tt.want[link]; !ok {
					t.Errorf("Crawl() link %v not found in %v", link, got)
				}
			}
			if tt.args.linkFound != nil {
				for range tt.wantLinkFound {
					select {
					case linkFromCallback := <-linkFoundCh:
						if _, ok := tt.want[linkFromCallback.String()]; !ok {
							t.Errorf("Crawl() linkFoundCallback executed with link %v not found in want list %v", linkFromCallback, tt.want)
						}
					case <-time.After(2 * time.Second):
						t.Error("Crawl() linkFound callback not called after waiting 2 seconds")
					}
				}
			}
			if tt.args.errorCallback != nil {
				select {
				case <-errorCallbackCh:
				case <-time.After(2 * time.Second):
					t.Error("Crawl() error callback not called after waiting 2 seconds")
				}
			}
		})
	}
}
