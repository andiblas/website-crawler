package crawler

import (
	"errors"
	"io"
	"net/url"
	"strings"
	"testing"

	"golang.org/x/net/context"

	"github.com/andiblas/website-crawler/pkg/fetcher"
)

const (
	htmlWithSingleLink      = `<a href="https://test.com"/>`
	htmlWithThreeLinks      = `<a href="https://test.com"/><a href="https://test.com/contact"/><a href="https://test.com/about-us"/>`
	htmlWithLinksDepthThree = `<a href="https://test.com"/><a href="https://test.com/depth1"/><a href="https://test.com/depth1/depth2"/><a href="https://test.com/depth1/depth2/depth3"/>`
)

type mockFetcher struct {
	webpageReader io.ReadCloser
	throwError    error
}

func (m mockFetcher) FetchWebpageContent(_ url.URL) (io.ReadCloser, error) {
	return m.webpageReader, m.throwError
}

func TestConcurrent_Crawl(t *testing.T) {
	canceledCtx, cancelFunc := context.WithCancel(context.Background())
	cancelFunc()
	testUrl, _ := url.Parse("https://test.com")
	type fields struct {
		fetcher fetcher.Fetcher
	}
	type args struct {
		ctx            context.Context
		urlToCrawl     url.URL
		recursionLimit int
		onNewLinkFound func(link url.URL)
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    map[string]bool
		wantErr bool
	}{
		{
			name: "crawls a page with a single link and returns it",
			fields: fields{fetcher: mockFetcher{
				webpageReader: io.NopCloser(strings.NewReader(htmlWithSingleLink)),
				throwError:    nil,
			}},
			args: args{
				ctx:            context.Background(),
				urlToCrawl:     *testUrl,
				recursionLimit: 1,
				onNewLinkFound: nil,
			},
			want: map[string]bool{
				"https://test.com": true,
			},
			wantErr: false,
		},
		{
			name: "crawls a page with three links and returns them",
			fields: fields{fetcher: mockFetcher{
				webpageReader: io.NopCloser(strings.NewReader(htmlWithThreeLinks)),
				throwError:    nil,
			}},
			args: args{
				ctx:            context.Background(),
				urlToCrawl:     *testUrl,
				recursionLimit: 2,
				onNewLinkFound: nil,
			},
			want: map[string]bool{
				"https://test.com":          true,
				"https://test.com/about-us": true,
				"https://test.com/contact":  true,
			},
			wantErr: false,
		},
		{
			name: "crawls a page returns an error",
			fields: fields{fetcher: mockFetcher{
				webpageReader: nil,
				throwError:    errors.New("error"),
			}},
			args: args{
				ctx:            context.Background(),
				urlToCrawl:     *testUrl,
				recursionLimit: 1,
				onNewLinkFound: nil,
			},
			want:    map[string]bool{},
			wantErr: true,
		},
		{
			name: "crawls gets interrupted",
			fields: fields{fetcher: mockFetcher{
				webpageReader: io.NopCloser(strings.NewReader(htmlWithSingleLink)),
				throwError:    nil,
			}},
			args: args{
				ctx:            canceledCtx,
				urlToCrawl:     *testUrl,
				recursionLimit: 2,
				onNewLinkFound: nil,
			},
			want:    map[string]bool{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewConcurrent(tt.fields.fetcher)
			got, err := c.Crawl(tt.args.ctx, tt.args.urlToCrawl, tt.args.recursionLimit, tt.args.onNewLinkFound)
			if (err != nil) != tt.wantErr {
				t.Errorf("Crawl() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != len(tt.want) {
				t.Errorf("Crawl() links len got %v want len %v", len(got), len(tt.want))
			}
			for _, link := range got {
				if _, ok := tt.want[link]; !ok {
					t.Errorf("Crawl() link %v not found in %v", link, got)
				}
			}
		})
	}
}
