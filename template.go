package sitetools

import (
	"bytes"
	"maps"
	"path"
	"text/template"
)

type TemplateTransformer struct {
	Components map[string]*Asset
	GlobalData map[string]any
}

type WrapperTemplate struct {
	Template       *Asset
	ChildBlockName string
}

func (t TemplateTransformer) Transform(asset *Asset) error {
	return t.transform(asset, nil)
}

func (t TemplateTransformer) TransformWithWrapper(asset *Asset, wrapperTemplate WrapperTemplate) error {
	return t.transform(asset, &wrapperTemplate)
}

func (t TemplateTransformer) transform(asset *Asset, wrapperTemplate *WrapperTemplate) error {
	if asset.Path == "/base_template.html" {
		return nil
	}

	var primarySource []byte
	templateMeta := map[string]any{
		"Global": t.GlobalData,
		"Asset":  asset.Meta,
	}

	if wrapperTemplate != nil {
		primarySource = wrapperTemplate.Template.Data
		templateMeta["WrapperTemplate"] = wrapperTemplate.Template.Meta
		maps.Copy(templateMeta, wrapperTemplate.Template.Meta)
	} else {
		primarySource = asset.Data
	}

	maps.Copy(templateMeta, asset.Meta)

	tmpl, err := template.New("").Parse(string(primarySource))
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

	if wrapperTemplate != nil {
		tmpl, err = tmpl.New(wrapperTemplate.ChildBlockName).Parse(string(asset.Data))
		if err != nil {
			return err
		}
	}

	buf := &bytes.Buffer{}
	tmpl.ExecuteTemplate(buf, "", templateMeta)

	asset.Data = buf.Bytes()
	return nil
}
