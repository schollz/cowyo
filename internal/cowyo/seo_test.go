package cowyo

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestPublishedPageSEO(t *testing.T) {
	t.Setenv(siteURLEnvironment, "https://cowyo.example")
	page := Page{
		Title:     "calm-cat",
		Text:      "A useful note\nwith enough detail to make a unique search preview.",
		Published: true,
	}
	request := httptest.NewRequest(http.MethodGet, "http://internal/paste", nil)

	seo, err := buildPageSEO(request, page)
	if err != nil {
		t.Fatalf("buildPageSEO() error = %v", err)
	}

	if seo.Title != "Calm Cat — Shared text on cowyo" {
		t.Errorf("Title = %q", seo.Title)
	}
	if seo.Description != "A useful note with enough detail to make a unique search preview. — Shared on cowyo, a minimalist online pastebin." {
		t.Errorf("Description = %q", seo.Description)
	}
	if seo.CanonicalURL != "https://cowyo.example/calm-cat" {
		t.Errorf("CanonicalURL = %q", seo.CanonicalURL)
	}
	if seo.SocialImageURL != "https://cowyo.example/static/og.jpg" {
		t.Errorf("SocialImageURL = %q", seo.SocialImageURL)
	}
	if seo.OpenGraphType != "article" {
		t.Errorf("OpenGraphType = %q, want article", seo.OpenGraphType)
	}
	if seo.Robots != robotsDirective(true) {
		t.Errorf("Robots = %q", seo.Robots)
	}

	var structuredData struct {
		Context string           `json:"@context"`
		Graph   []map[string]any `json:"@graph"`
	}
	if err := json.Unmarshal([]byte(seo.JSONLD), &structuredData); err != nil {
		t.Fatalf("JSON-LD is invalid: %v\n%s", err, seo.JSONLD)
	}
	if structuredData.Context != "https://schema.org" {
		t.Errorf("@context = %q", structuredData.Context)
	}
	if len(structuredData.Graph) != 6 {
		t.Fatalf("@graph entries = %d, want 6", len(structuredData.Graph))
	}
	if structuredData.Graph[0]["@type"] != "WebSite" {
		t.Errorf("first @graph type = %v", structuredData.Graph[0]["@type"])
	}
	if structuredData.Graph[1]["@type"] != "WebApplication" {
		t.Errorf("second @graph type = %v", structuredData.Graph[1]["@type"])
	}
	if structuredData.Graph[5]["@type"] != "DigitalDocument" {
		t.Errorf("published entity type = %v", structuredData.Graph[5]["@type"])
	}
}

func TestBuildLandingSEOUsesRootCanonicalAndWebsiteMetadata(t *testing.T) {
	t.Setenv(siteURLEnvironment, "https://cowyo.example")
	request := httptest.NewRequest(http.MethodGet, "http://internal.example/", nil)

	seo, err := buildLandingSEO(request)
	if err != nil {
		t.Fatalf("buildLandingSEO() error = %v", err)
	}

	if seo.Title != landingTitle {
		t.Errorf("Title = %q, want %q", seo.Title, landingTitle)
	}
	if seo.CanonicalURL != "https://cowyo.example/" {
		t.Errorf("CanonicalURL = %q, want root URL", seo.CanonicalURL)
	}
	if seo.OpenGraphType != "website" {
		t.Errorf("OpenGraphType = %q, want website", seo.OpenGraphType)
	}
	if seo.Robots != robotsDirective(true) {
		t.Errorf("Robots = %q, want indexable", seo.Robots)
	}
	if !seo.Published {
		t.Error("landing SEO is not marked published")
	}
	if strings.Contains(seo.JSONLD, "DigitalDocument") {
		t.Error("landing structured data describes a paste document")
	}
}

func TestBuildAboutSEOUsesAboutCanonicalAndWebsiteMetadata(t *testing.T) {
	t.Setenv(siteURLEnvironment, "https://cowyo.example")
	request := httptest.NewRequest(http.MethodGet, "http://internal.example/about", nil)

	seo, err := buildAboutSEO(request)
	if err != nil {
		t.Fatalf("buildAboutSEO() error = %v", err)
	}

	if seo.Title != aboutTitle {
		t.Errorf("Title = %q, want %q", seo.Title, aboutTitle)
	}
	if seo.Description != aboutDescription {
		t.Errorf("Description = %q, want %q", seo.Description, aboutDescription)
	}
	if seo.CanonicalURL != "https://cowyo.example/about" {
		t.Errorf("CanonicalURL = %q, want about URL", seo.CanonicalURL)
	}
	if seo.Robots != robotsDirective(true) || !seo.Published {
		t.Error("about SEO is not indexable")
	}
	if strings.Contains(seo.JSONLD, "DigitalDocument") {
		t.Error("about structured data describes a paste document")
	}
}

func TestUnpublishedSEOUsesGenericCopy(t *testing.T) {
	const secret = "do not put this secret in metadata"
	request := httptest.NewRequest(http.MethodGet, "https://cowyo.example/private-note", nil)

	seo, err := buildPageSEO(request, Page{
		Title: "private-note",
		Text:  secret,
	})
	if err != nil {
		t.Fatalf("buildPageSEO() error = %v", err)
	}

	if seo.Description != siteDescription {
		t.Errorf("Description = %q, want generic site description", seo.Description)
	}
	if strings.Contains(seo.Description, secret) || strings.Contains(seo.JSONLD, secret) {
		t.Fatal("unpublished paste text leaked into metadata")
	}
	if seo.OpenGraphType != "website" {
		t.Errorf("OpenGraphType = %q, want website", seo.OpenGraphType)
	}
	if seo.Robots != robotsDirective(false) {
		t.Errorf("Robots = %q", seo.Robots)
	}
}

func TestEncryptedPublishedSEODoesNotExposeCiphertext(t *testing.T) {
	const encrypted = encryptedBlockStart + "\n{\"data\":\"secret-ciphertext\"}\n" + encryptedBlockEnd
	description := pageSEODescription(Page{
		Text:      encrypted,
		Published: true,
	})

	if !strings.Contains(description, "encrypted text paste") {
		t.Errorf("Description = %q, want encrypted-paste description", description)
	}
	if strings.Contains(description, "secret-ciphertext") {
		t.Fatal("encrypted payload leaked into the description")
	}
}

func TestSEOMetadataIsEscapedAndBounded(t *testing.T) {
	page := Page{
		Title:     `notes"><script>alert(1)</script>`,
		Text:      strings.Repeat("shareable text ", 30),
		Published: true,
	}
	request := httptest.NewRequest(http.MethodGet, "https://cowyo.example/notes", nil)

	seo, err := buildPageSEO(request, page)
	if err != nil {
		t.Fatalf("buildPageSEO() error = %v", err)
	}

	if strings.Contains(seo.Title, "<script>") {
		t.Fatalf("Title was not escaped: %q", seo.Title)
	}
	if utf8.RuneCountInString(pageSEODescription(page)) > 160 {
		t.Errorf("description has %d runes, want at most 160", utf8.RuneCountInString(pageSEODescription(page)))
	}
	if utf8.RuneCountInString(pageSEOTitle(page)) > 60 {
		t.Errorf("title has %d runes, want at most 60", utf8.RuneCountInString(pageSEOTitle(page)))
	}
	if strings.Contains(seo.JSONLD, "</script>") {
		t.Fatal("JSON-LD contains an unescaped script terminator")
	}
}

func TestConfiguredSiteURL(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "http://request.example/page", nil)
	request.Header.Set("X-Forwarded-Proto", "https")

	for _, tt := range []struct {
		name       string
		configured string
		want       string
	}{
		{
			name:       "configured origin",
			configured: "https://canonical.example/",
			want:       "https://canonical.example/page",
		},
		{
			name:       "rejects path",
			configured: "https://canonical.example/subpath",
			want:       "https://request.example/page",
		},
		{
			name:       "rejects credentials",
			configured: "https://user:pass@canonical.example",
			want:       "https://request.example/page",
		},
		{
			name:       "rejects non-http scheme",
			configured: "javascript:alert(1)",
			want:       "https://request.example/page",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(siteURLEnvironment, tt.configured)
			if got := postedDocumentURL(request, "page"); got != tt.want {
				t.Errorf("postedDocumentURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBrowserPageIncludesCompleteMetadata(t *testing.T) {
	t.Setenv(siteURLEnvironment, "https://cowyo.example")
	setUpHandlerTest(t, Page{
		Title:     "search-friendly",
		Text:      "A useful published paste.",
		Published: true,
	})

	request := httptest.NewRequest(http.MethodGet, "/search-friendly", nil)
	request.Header.Set("User-Agent", "Mozilla/5.0")
	response := httptest.NewRecorder()
	if err := handle(response, request); err != nil {
		t.Fatalf("handle() error = %v", err)
	}

	body := response.Body.String()
	for description, marker := range map[string]string{
		"title":              "<title>Search Friendly — Shared text on cowyo</title>",
		"description":        `name="description"`,
		"canonical":          `rel="canonical" href="https://cowyo.example/search-friendly"`,
		"Open Graph title":   `property="og:title"`,
		"Open Graph image":   `property="og:image" content="https://cowyo.example/static/og.jpg"`,
		"X card":             `name="twitter:card" content="summary_large_image"`,
		"structured data":    `type="application/ld+json"`,
		"published document": `"@type":"DigitalDocument"`,
	} {
		if !strings.Contains(body, marker) {
			t.Errorf("browser page does not contain %s marker %q", description, marker)
		}
	}
	if got := response.Header().Get("Link"); got != `<https://cowyo.example/search-friendly>; rel="canonical"` {
		t.Errorf("Link header = %q", got)
	}
	if got := response.Header().Get("Content-Language"); got != "en-US" {
		t.Errorf("Content-Language = %q, want en-US", got)
	}
}

func TestSocialPreviewImageIsServed(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/static/og.jpg", nil)
	response := httptest.NewRecorder()

	if err := handle(response, request); err != nil {
		t.Fatalf("handle() error = %v", err)
	}
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if got := response.Header().Get("Content-Type"); got != "image/jpeg" {
		t.Errorf("Content-Type = %q, want image/jpeg", got)
	}
	if response.Body.Len() < 50_000 {
		t.Errorf("social preview image is only %d bytes", response.Body.Len())
	}
}

func TestVendoredLogoAndSocialPreview(t *testing.T) {
	tests := []struct {
		path       string
		wantWidth  int
		wantHeight int
		wantSHA256 string
	}{
		{
			path:       "static/logo.jpg",
			wantWidth:  1515,
			wantHeight: 668,
			wantSHA256: "8d47ed0095ae8ac239ad8e08942761c7cef04f9ce769625fc40644f3ddd596cb",
		},
		{
			path:       "static/og.jpg",
			wantWidth:  1200,
			wantHeight: 630,
			wantSHA256: "2320ff0be4ef7831d40e4f3b378bd75b22025657c0a94ea6e20e2a64862e5595",
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			data, err := fs.ReadFile(siteContent, tt.path)
			if err != nil {
				t.Fatalf("read asset: %v", err)
			}
			sum := sha256.Sum256(data)
			if got := fmt.Sprintf("%x", sum); got != tt.wantSHA256 {
				t.Errorf("SHA-256 = %s, want %s", got, tt.wantSHA256)
			}

			config, format, err := image.DecodeConfig(bytes.NewReader(data))
			if err != nil {
				t.Fatalf("decode image config: %v", err)
			}
			if format != "jpeg" {
				t.Errorf("format = %q, want jpeg", format)
			}
			if config.Width != tt.wantWidth || config.Height != tt.wantHeight {
				t.Errorf(
					"dimensions = %dx%d, want %dx%d",
					config.Width,
					config.Height,
					tt.wantWidth,
					tt.wantHeight,
				)
			}
		})
	}
}
