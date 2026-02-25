package sitetools

import (
	"bytes"
	"encoding/xml"
	"fmt"
)

func xmlEscape(value string) string {
	var buf bytes.Buffer
	_ = xml.EscapeText(&buf, []byte(value))
	return buf.String()
}

func (build *Build) AddSitemap(url string, filters ...Filter) error {
	if len(build.Assets) == 0 {
		return nil
	}

	data := make([]byte, 0)
	data = append(data, []byte(`<?xml version="1.0" encoding="UTF-8"?>`)...)
	data = append(data, []byte(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)...)

	filters = append(filters, WithoutMeta("SitemapExclude"))

	for _, asset := range build.Assets.Filter(filters...) {
		data = append(data, []byte("<url>")...)
		data = append(data, []byte("<loc>"+xmlEscape(url+asset.Path)+"</loc>")...)

		acceptableModifiedKeys := []string{"SitemapLastModified", "LastModified"}
		for _, key := range acceptableModifiedKeys {
			if modified, ok := asset.Meta[key]; ok {
				if modifiedStr, ok := modified.(string); ok {
					data = append(data, []byte("<lastmod>"+xmlEscape(modifiedStr)+"</lastmod>")...)
					break
				}
			}
		}

		acceptablePriorityKeys := []string{"SitemapPriority", "Priority"}
		for _, key := range acceptablePriorityKeys {
			if priority, ok := asset.Meta[key]; ok {
				switch p := priority.(type) {
				case float32:
					data = append(data, []byte("<priority>"+xmlEscape(fmt.Sprintf("%.1f", float64(p)))+"</priority>")...)
					break
				case int:
					data = append(data, []byte("<priority>"+xmlEscape(fmt.Sprintf("%.1f", float64(p)))+"</priority>")...)
					break
				case int32:
					data = append(data, []byte("<priority>"+xmlEscape(fmt.Sprintf("%.1f", float64(p)))+"</priority>")...)
					break
				case int64:
					data = append(data, []byte("<priority>"+xmlEscape(fmt.Sprintf("%.1f", float64(p)))+"</priority>")...)
					break
				case float64:
					data = append(data, []byte("<priority>"+xmlEscape(fmt.Sprintf("%.1f", p))+"</priority>")...)
					break
				case string:
					data = append(data, []byte("<priority>"+xmlEscape(p)+"</priority>")...)
					break
				}
			}
		}

		acceptableChangeFreqKeys := []string{"SitemapChangeFreq", "ChangeFreq"}
		for _, key := range acceptableChangeFreqKeys {
			if changeFreq, ok := asset.Meta[key]; ok {
				if changeFreqStr, ok := changeFreq.(string); ok {
					data = append(data, []byte("<changefreq>"+xmlEscape(changeFreqStr)+"</changefreq>")...)
					break
				}
			}
		}

		data = append(data, []byte("</url>")...)
	}

	data = append(data, []byte(`</urlset>`)...)

	sitemap := &Asset{
		Path: "/sitemap.xml",
		Data: data,
		Meta: map[string]any{"ContentType": "application/xml"},
	}

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
