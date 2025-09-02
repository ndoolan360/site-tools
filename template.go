package sitetools

import (
	"bytes"
	"fmt"
	"maps"
	"path"
	"text/template"
)

type TemplateTransformer struct {
	Components map[string]*Asset
	Global     map[string]any
}

func (t TemplateTransformer) Transform(asset *Asset) error {
	if asset.Meta["Global"] != nil {
		return fmt.Errorf("asset meta cannot contain reserved key 'Global'")
	}

	templateMeta := map[string]any{"Global": t.Global}
	maps.Copy(templateMeta, asset.Meta)

	tmpl, err := template.New("").Parse(string(asset.Data))
	if err != nil {
		return err
	}

	for name, component := range t.Components {
		if path.Ext(component.Path) != path.Ext(asset.Path) {
			continue
		}
		tmpl, err = tmpl.New(name).Parse(string(component.Data))
		if err != nil {
			return err
		}
	}

	buf := &bytes.Buffer{}
	if err := tmpl.Lookup("").Option("missingkey=zero").Execute(buf, templateMeta); err != nil {
		return err
	}

	asset.Data = buf.Bytes()

	return nil
}
