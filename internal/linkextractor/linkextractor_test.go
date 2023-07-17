package linkextractor

import (
	"net/url"
	"reflect"
	"testing"
)

const (
	htmlWithLinks         = `<a href="https://test.com"></a><a href="https://google.com"></a>`
	htmlWithNoLinks       = `<body><p>I have no links</p></body>`
	htmlWithRepeatedLinks = `<a href="https://test.com"/><a href="https://google.com"/><a href="https://test.com"/>`
)

func TestExtract(t *testing.T) {
	testUrl, _ := url.Parse("https://test.com")
	assertUrl, _ := url.Parse("https://test.com")
	type args struct {
		webpageURL     url.URL
		webpageContent string
	}
	tests := []struct {
		name    string
		args    args
		want    []url.URL
		wantErr bool
	}{
		{
			name: "extracts links from same domain from html that has links",
			args: args{
				webpageURL:     *testUrl,
				webpageContent: htmlWithLinks,
			},
			want: []url.URL{
				*assertUrl,
			},
			wantErr: false,
		},
		{
			name: "extracts no links from a link-less html",
			args: args{
				webpageURL:     *testUrl,
				webpageContent: htmlWithNoLinks,
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "extracts links without repeating",
			args: args{
				webpageURL:     *testUrl,
				webpageContent: htmlWithRepeatedLinks,
			},
			want: []url.URL{
				*assertUrl,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Extract(tt.args.webpageURL, tt.args.webpageContent)
			if (err != nil) != tt.wantErr {
				t.Errorf("Extract() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Extract() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalize(t *testing.T) {
	type args struct {
		urlToNormalize string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "removes www.",
			args: args{
				urlToNormalize: "https://www.google.com",
			},
			want: "https://google.com",
		},
		{
			name: "removes trailing /",
			args: args{
				urlToNormalize: "https://google.com/",
			},
			want: "https://google.com",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputUrl, err := url.Parse(tt.args.urlToNormalize)
			if err != nil {
				t.Errorf("input url is not valid. %v", err)
			}
			if got := Normalize(*inputUrl); !reflect.DeepEqual(got.String(), tt.want) {
				t.Errorf("Normalize() = %v, want %v", got, tt.want)
			}
		})
	}
}
