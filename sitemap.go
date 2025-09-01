package sitetools

func (build *Build) AddSitemap() (*Asset, error) {
	if len(build.Assets) == 0 {
		return nil, nil
	}

	data := make([]byte, 0)
	data = append(data, []byte(`<?xml version="1.0" encoding="UTF-8"?>`+"\n")...)
	data = append(data, []byte(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`+"\n")...)

	for _, asset := range build.Assets.Filter(
		WithExtensions(".html"),
		WithoutMeta("Draft"),
		WithoutMeta("SitemapExclude"),
	) {
		data = append(data, []byte("  <url>\n")...)
		data = append(data, []byte("    <loc>https://ndoolan.com"+asset.Path+"</loc>\n")...)
		acceptableModifiedKeys := []string{"LastModified", "Modified", "Date"}
		for _, key := range acceptableModifiedKeys {
			if modified, ok := asset.Meta[key]; ok {
				if modifiedStr, ok := modified.(string); ok {
					data = append(data, []byte("    <lastmod>"+modifiedStr+"</lastmod>\n")...)
					break
				}
			}
		}
		data = append(data, []byte("  </url>\n")...)
	}

	data = append(data, []byte(`</urlset>`)...)

	sitemap := &Asset{
		Path: "sitemap.xml",
		Data: data,
		Meta: map[string]any{"ContentType": "application/xml"},
	}

	min := MinifyTransformer{}
	min.Transform(sitemap)

	return sitemap, nil
}

func (build *Build) AddRobotsTxt(additionalLines ...string) (*Asset, error) {
	if len(build.Assets) == 0 {
		return nil, nil
	}

	data := []byte("User-agent: *\nDisallow: /\n")
	for _, line := range additionalLines {
		data = append(data, []byte(line+"\n")...)
	}

	robots := &Asset{
		Path: "robots.txt",
		Data: data,
		Meta: map[string]any{"ContentType": "text/plain"},
	}
	return robots, nil
}
