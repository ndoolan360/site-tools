package sitetools

import "testing"

func TestAddSitemap(t *testing.T) {
	build := &Build{
		Assets: Assets{
			&Asset{Path: "/index.html"},
			&Asset{Path: "/about.html"},
			&Asset{Path: "/contact.html"},
			&Asset{Path: "/styles.css"},
		},
	}

	err := build.AddSitemap("https://test.com")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	sitemap := build.Assets.Filter(WithPath("/sitemap.xml"))[0]
	if sitemap == nil {
		t.Fatal("Expected sitemap asset, got nil")
	}

	expectedData := `<?xml version="1.0" encoding="UTF-8"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"><url><loc>https://test.com/index.html</loc></url><url><loc>https://test.com/about.html</loc></url><url><loc>https://test.com/contact.html</loc></url><url><loc>https://test.com/styles.css</loc></url></urlset>`
	if string(sitemap.Data) != expectedData {
		t.Errorf("Sitemap data does not match expected.\nGot:\n%s\nExpected:\n%s", string(sitemap.Data), expectedData)
	}

	if sitemap.Path != "/sitemap.xml" {
		t.Errorf("Expected sitemap path to be 'sitemap.xml', got '%s'", sitemap.Path)
	}

	contentType, ok := sitemap.Meta["ContentType"].(string)
	if !ok || contentType != "application/xml" {
		t.Errorf("Expected ContentType to be 'application/xml', got '%v'", sitemap.Meta["ContentType"])
	}
}

func TestAddSitemap_WithExclusion(t *testing.T) {
	build := &Build{
		Assets: Assets{
			&Asset{Path: "/index.html", Meta: map[string]any{"SitemapPriority": 1.0}},
			&Asset{Path: "/important.html", Meta: map[string]any{"SitemapPriority": "0.8"}},
			&Asset{Path: "/about.html", Meta: map[string]any{"LastModified": "2025-08-02"}},
			&Asset{Path: "/contact.html", Meta: map[string]any{"SitemapExclude": false}},
			&Asset{Path: "/private.html", Meta: map[string]any{"SitemapExclude": true}},
			&Asset{Path: "/styles.css", Meta: map[string]any{"SitemapChangeFreq": "never"}},
		},
	}

	err := build.AddSitemap("https://test.com")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	sitemap := build.Assets.Filter(WithPath("/sitemap.xml"))[0]
	if sitemap == nil {
		t.Fatal("Expected sitemap asset, got nil")
	}

	expectedData := `<?xml version="1.0" encoding="UTF-8"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"><url><loc>https://test.com/index.html</loc><priority>1.0</priority></url><url><loc>https://test.com/important.html</loc><priority>0.8</priority></url><url><loc>https://test.com/about.html</loc><lastmod>2025-08-02</lastmod></url><url><loc>https://test.com/contact.html</loc></url><url><loc>https://test.com/styles.css</loc><changefreq>never</changefreq></url></urlset>`
	if string(sitemap.Data) != expectedData {
		t.Errorf("Sitemap data does not match expected.\nGot:\n%s\nExpected:\n%s", string(sitemap.Data), expectedData)
	}
}

func TestAddRobotsTxt(t *testing.T) {
	build := &Build{
		Assets: Assets{
			&Asset{Path: "/index.html"},
		},
	}

	err := build.AddRobotsTxt(
		"Allow: /",
		"Sitemap: https://example.com/sitemap.xml",
	)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	robots := build.Assets.Filter(WithPath("/robots.txt"))[0]
	if robots == nil {
		t.Fatal("Expected robots.txt asset, got nil")
	}

	expectedData := `User-agent: *
Disallow: /
Allow: /
Sitemap: https://example.com/sitemap.xml
`

	if string(robots.Data) != expectedData {
		t.Errorf("Robots.txt data does not match expected.\nGot:\n%s\nExpected:\n%s", string(robots.Data), expectedData)
	}

	if robots.Path != "/robots.txt" {
		t.Errorf("Expected robots.txt path to be 'robots.txt', got '%s'", robots.Path)
	}

	contentType, ok := robots.Meta["ContentType"].(string)
	if !ok || contentType != "text/plain" {
		t.Errorf("Expected ContentType to be 'text/plain', got '%v'", robots.Meta["ContentType"])
	}
}
