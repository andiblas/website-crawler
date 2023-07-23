package crawler

import (
	"errors"
	"net/url"
	"testing"

	"golang.org/x/net/context"

	"github.com/andiblas/website-crawler/pkg/fetcher"
)

const (
	htmlWithSingleLink = `<a href="https://test.com"/>`
	htmlWithThreeLinks = `<a href="https://test.com"/><a href="https://test.com/contact"/><a href="https://test.com/about-us"/>`
)

type mockFetcher struct {
	webpageContent string
	throwError     error
}

func (m mockFetcher) FetchWebpageContent(_ url.URL) (string, error) {
	return m.webpageContent, m.throwError
}

func TestConcurrent_Crawl(t *testing.T) {
	testUrl, _ := url.Parse("https://test.com")
	type fields struct {
		fetcher fetcher.Fetcher
	}
	type args struct {
		ctx        context.Context
		urlToCrawl url.URL
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
				webpageContent: htmlWithSingleLink,
				throwError:     nil,
			}},
			args: args{
				ctx:        context.Background(),
				urlToCrawl: *testUrl,
			},
			want: map[string]bool{
				"https://test.com": true,
			},
			wantErr: false,
		},
		{
			name: "crawls a page with three links and returns them",
			fields: fields{fetcher: mockFetcher{
				webpageContent: htmlWithThreeLinks,
				throwError:     nil,
			}},
			args: args{
				ctx:        context.Background(),
				urlToCrawl: *testUrl,
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
				webpageContent: "",
				throwError:     errors.New("error"),
			}},
			args: args{
				ctx:        context.Background(),
				urlToCrawl: *testUrl,
			},
			want:    map[string]bool{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Concurrent{
				fetcher: tt.fields.fetcher,
			}
			got, err := c.Crawl(tt.args.ctx, tt.args.urlToCrawl)
			if (err != nil) != tt.wantErr {
				t.Errorf("Crawl() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			for _, link := range got {
				if _, ok := tt.want[link]; !ok {
					t.Errorf("Crawl() link %v not found in %v", link, got)
				}
			}
		})
	}
}
