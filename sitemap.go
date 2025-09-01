package sitetools

import "fmt"

func (build *Build) AddSitemap(url string, filters ...Filter) error {
	if len(build.Assets) == 0 {
		return nil
	}

	data := make([]byte, 0)
	data = append(data, []byte(`<?xml version="1.0" encoding="UTF-8"?>`+"\n")...)
	data = append(data, []byte(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`+"\n")...)

	filters = append(filters, WithoutMeta("SitemapExclude"))

	for _, asset := range build.Assets.Filter(filters...) {
		data = append(data, []byte("  <url>\n")...)
		data = append(data, []byte("    <loc>"+url+asset.Path+"</loc>\n")...)

		acceptableModifiedKeys := []string{"SitemapLastModified", "LastModified"}
		for _, key := range acceptableModifiedKeys {
			if modified, ok := asset.Meta[key]; ok {
				if modifiedStr, ok := modified.(string); ok {
					data = append(data, []byte("    <lastmod>"+modifiedStr+"</lastmod>\n")...)
					break
				}
			}
		}

		acceptablePriorityKeys := []string{"SitemapPriority", "Priority"}
		for _, key := range acceptablePriorityKeys {
			if priority, ok := asset.Meta[key]; ok {
				switch priority.(type) {
				case float32, float64, int, int32, int64:
					data = append(data, []byte("    <priority>"+fmt.Sprintf("%.1f", priority)+"</priority>\n")...)
					break
				case string:
					data = append(data, []byte("    <priority>"+priority.(string)+"</priority>\n")...)
					break
				}
			}
		}

		acceptableChangeFreqKeys := []string{"SitemapChangeFreq", "ChangeFreq"}
		for _, key := range acceptableChangeFreqKeys {
			if changeFreq, ok := asset.Meta[key]; ok {
				if changeFreqStr, ok := changeFreq.(string); ok {
					data = append(data, []byte("    <changefreq>"+changeFreqStr+"</changefreq>\n")...)
					break
				}
			}
		}

		data = append(data, []byte("  </url>\n")...)
	}

	data = append(data, []byte(`</urlset>`)...)

	sitemap := &Asset{
		Path: "/sitemap.xml",
		Data: data,
		Meta: map[string]any{"ContentType": "application/xml"},
	}

	min := MinifyTransformer{}
	min.Transform(sitemap)

	build.Assets = append(build.Assets, sitemap)

	return nil
}

func (build *Build) AddRobotsTxt(additionalLines ...string) error {
	if len(build.Assets) == 0 {
		return nil
	}

	data := []byte("User-agent: *\nDisallow: /\n")
	for _, line := range additionalLines {
		data = append(data, []byte(line+"\n")...)
	}

	robots := &Asset{
		Path: "/robots.txt",
		Data: data,
		Meta: map[string]any{"ContentType": "text/plain"},
	}

	build.Assets = append(build.Assets, robots)

	return nil
}
