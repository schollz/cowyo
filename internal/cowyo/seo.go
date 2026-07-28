package cowyo

import (
	"encoding/json"
	"html"
	"net/http"
	"strings"
	"unicode/utf8"
)

const (
	siteName        = "cowyo"
	siteDescription = "Create and share text instantly with cowyo, a minimalist online pastebin with live autosave, collaboration, browser-side encryption, and self-destructing pastes."
	socialImageAlt  = "cowyo logo — a cow beside a speech bubble saying yo"
)

type seoTemplateData struct {
	Title          string
	Description    string
	Robots         string
	CanonicalURL   string
	SiteURL        string
	SitemapURL     string
	SocialImageURL string
	OpenGraphType  string
	Published      bool
	JSONLD         string
}

type rawPageSEO struct {
	title          string
	description    string
	robots         string
	canonicalURL   string
	siteURL        string
	sitemapURL     string
	socialImageURL string
	logoURL        string
	openGraphType  string
}

func buildPageSEO(r *http.Request, page Page) (seoTemplateData, error) {
	raw := rawPageSEO{
		title:          pageSEOTitle(page),
		description:    pageSEODescription(page),
		robots:         robotsDirective(page.Published),
		canonicalURL:   postedDocumentURL(r, page.Title),
		siteURL:        postedDocumentURL(r, ""),
		sitemapURL:     postedDocumentURL(r, "sitemap.xml"),
		socialImageURL: postedDocumentURL(r, "static/og.jpg"),
		logoURL:        postedDocumentURL(r, "static/logo.jpg"),
		openGraphType:  "website",
	}
	if page.Published {
		raw.openGraphType = "article"
	}

	jsonLD, err := structuredDataJSON(raw, page)
	if err != nil {
		return seoTemplateData{}, err
	}

	return seoTemplateData{
		Title:          html.EscapeString(raw.title),
		Description:    html.EscapeString(raw.description),
		Robots:         raw.robots,
		CanonicalURL:   html.EscapeString(raw.canonicalURL),
		SiteURL:        html.EscapeString(raw.siteURL),
		SitemapURL:     html.EscapeString(raw.sitemapURL),
		SocialImageURL: html.EscapeString(raw.socialImageURL),
		OpenGraphType:  raw.openGraphType,
		Published:      page.Published,
		JSONLD:         string(jsonLD),
	}, nil
}

func pageSEOTitle(page Page) string {
	name := humanizePageTitle(page.Title)
	if name == "" {
		name = "Untitled paste"
	}

	suffix := " — cowyo paste"
	if page.Published {
		suffix = " — Shared text on cowyo"
	}
	return truncateText(name, 60-utf8.RuneCountInString(suffix)) + suffix
}

func pageSEODescription(page Page) string {
	if !page.Published {
		return siteDescription
	}

	plainText := encryptedBlockPattern.ReplaceAllString(page.Text, " ")
	excerpt := strings.Join(strings.Fields(plainText), " ")
	if excerpt != "" {
		const suffix = " — Shared on cowyo, a minimalist online pastebin."
		return truncateText(
			excerpt,
			160-utf8.RuneCountInString(suffix),
		) + suffix
	}
	if encryptedBlockPattern.MatchString(page.Text) {
		return "An encrypted text paste shared on cowyo, a minimalist online pastebin with browser-side encryption."
	}
	return "A published text paste shared on cowyo, a minimalist online pastebin for creating and sharing text instantly."
}

func humanizePageTitle(title string) string {
	words := strings.FieldsFunc(title, func(r rune) bool {
		return r == '-' || r == '_'
	})
	for i, word := range words {
		runes := []rune(word)
		if len(runes) == 0 {
			continue
		}
		if runes[0] >= 'a' && runes[0] <= 'z' {
			runes[0] -= 'a' - 'A'
		}
		words[i] = string(runes)
	}
	return strings.Join(words, " ")
}

func truncateText(value string, maxRunes int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	if maxRunes <= 1 {
		return "…"
	}

	truncated := strings.TrimSpace(string(runes[:maxRunes-1]))
	if boundary := strings.LastIndexAny(truncated, " \t\n\r"); boundary >= maxRunes/2 {
		truncated = truncated[:boundary]
	}
	truncated = strings.TrimRight(truncated, " ,.;:!?-—")
	return truncated + "…"
}

func robotsDirective(published bool) string {
	if published {
		return "index, follow, max-snippet:-1, max-image-preview:large, max-video-preview:-1"
	}
	return "noindex, nofollow, noarchive, nosnippet, noimageindex"
}

func structuredDataJSON(seo rawPageSEO, page Page) ([]byte, error) {
	websiteID := seo.siteURL + "#website"
	webAppID := seo.siteURL + "#webapp"
	webPageID := seo.canonicalURL + "#webpage"
	imageID := seo.socialImageURL + "#primaryimage"
	logoID := seo.logoURL + "#logo"

	webPage := map[string]any{
		"@type":       "WebPage",
		"@id":         webPageID,
		"url":         seo.canonicalURL,
		"name":        seo.title,
		"description": seo.description,
		"inLanguage":  "en-US",
		"isPartOf": map[string]string{
			"@id": websiteID,
		},
		"about": map[string]string{
			"@id": webAppID,
		},
		"primaryImageOfPage": map[string]string{
			"@id": imageID,
		},
	}

	graph := []any{
		map[string]any{
			"@type":         "WebSite",
			"@id":           websiteID,
			"url":           seo.siteURL,
			"name":          siteName,
			"alternateName": "cowyo2",
			"description":   siteDescription,
			"inLanguage":    "en-US",
			"image": map[string]string{
				"@id": logoID,
			},
		},
		map[string]any{
			"@type":               "WebApplication",
			"@id":                 webAppID,
			"url":                 seo.siteURL,
			"name":                siteName,
			"description":         siteDescription,
			"applicationCategory": "UtilitiesApplication",
			"operatingSystem":     "Any",
			"browserRequirements": "Requires JavaScript and a modern web browser.",
			"isAccessibleForFree": true,
			"image": map[string]string{
				"@id": logoID,
			},
			"featureList": []string{
				"Instant text sharing with a memorable URL",
				"Live autosave and real-time collaborative editing",
				"Optional browser-side encryption",
				"Password-protected editing",
				"Published or unlisted pastes",
				"Self-destructing pastes",
				"Plaintext curl access",
			},
			"offers": map[string]any{
				"@type":         "Offer",
				"price":         "0",
				"priceCurrency": "USD",
			},
		},
		map[string]any{
			"@type":      "ImageObject",
			"@id":        logoID,
			"url":        seo.logoURL,
			"contentUrl": seo.logoURL,
			"caption":    socialImageAlt,
			"width":      1515,
			"height":     668,
		},
		map[string]any{
			"@type":      "ImageObject",
			"@id":        imageID,
			"url":        seo.socialImageURL,
			"contentUrl": seo.socialImageURL,
			"caption":    socialImageAlt,
			"width":      1200,
			"height":     630,
		},
		webPage,
	}

	if page.Published {
		pasteID := seo.canonicalURL + "#paste"
		webPage["mainEntity"] = map[string]string{"@id": pasteID}
		graph = append(graph, map[string]any{
			"@type":               "DigitalDocument",
			"@id":                 pasteID,
			"url":                 seo.canonicalURL,
			"name":                humanizePageTitle(page.Title),
			"description":         seo.description,
			"encodingFormat":      "text/plain",
			"inLanguage":          "en-US",
			"isAccessibleForFree": true,
			"isPartOf": map[string]string{
				"@id": websiteID,
			},
		})
	}

	return json.Marshal(map[string]any{
		"@context": "https://schema.org",
		"@graph":   graph,
	})
}
