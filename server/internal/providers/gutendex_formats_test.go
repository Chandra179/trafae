package providers

import (
	"slices"
	"testing"
)

func TestFormatURLsKeepsBookContentInStableOrder(t *testing.T) {
	formats := map[string]string{
		"text/html; charset=utf-8":       "https://books.test/b.html",
		"application/xhtml+xml":          "https://books.test/a.xhtml",
		"application/pdf":                "https://books.test/book.pdf",
		"text/plain; charset=us-ascii":   "https://books.test/book.txt",
		"application/epub+zip":           "https://books.test/book.epub",
		"application/x-mobipocket-ebook": "http://books.test/book.mobi",
		"image/jpeg":                     "https://books.test/cover.jpg",
		"application/rdf+xml":            "https://books.test/book.rdf",
		"application/zip":                "https://books.test/book.zip",
		"application/octet-stream":       "https://books.test/unknown",
		"text/html; charset=ascii":       "https://books.test/b.html",
		"text/plain; charset=utf-8":      " https://books.test/book.txt ",
	}
	want := []string{
		"https://books.test/a.xhtml", "https://books.test/b.html",
		"https://books.test/book.pdf", "https://books.test/book.txt",
		"https://books.test/book.epub", "http://books.test/book.mobi",
	}
	for i := 0; i < 30; i++ {
		if got := formatURLs(formats); !slices.Equal(got, want) {
			t.Fatalf("formatURLs = %v, want %v", got, want)
		}
	}
}

func TestFormatURLsRejectsInvalidURLsAndNonBookFormats(t *testing.T) {
	for _, value := range []string{
		"", "   ", "/book.html", "//books.test/book.html", "javascript:alert(1)",
		"file:///book.html", "https:///book.html", "https://", "https://books.test/%zz",
		"https://books.test:invalid/book.html",
	} {
		t.Run(value, func(t *testing.T) {
			if got := formatURLs(map[string]string{"text/html": value}); len(got) != 0 {
				t.Fatalf("formatURLs(%q) = %v, want no links", value, got)
			}
		})
	}
	for _, formats := range []map[string]string{
		nil,
		{"image/png": "https://books.test/cover.png"},
		{"application/rdf+xml": "https://books.test/book.rdf"},
		{"text/html; invalid": "https://books.test/book.html"},
	} {
		if got := formatURLs(formats); len(got) != 0 {
			t.Fatalf("formatURLs = %v, want no links", got)
		}
	}
}
