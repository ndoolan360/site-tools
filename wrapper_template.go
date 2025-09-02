package sitetools

import (
	"maps"
)

type WrapperTemplateTransformer struct {
	TemplateTransformer
	WrapperTemplate
}

type WrapperTemplate struct {
	Template       *Asset
	ChildBlockName string
}

func (t WrapperTemplateTransformer) Transform(asset *Asset) error {
	wrapperComponents := map[string]*Asset{t.ChildBlockName: asset}
	maps.Copy(wrapperComponents, t.Components)

	wrappedAsset := Asset{
		Path: asset.Path,
		Data: t.Template.Data,
		Meta: t.Template.Meta,
	}
	maps.Copy(wrappedAsset.Meta, asset.Meta)

	transformer := TemplateTransformer{
		Global:     t.Global,
		Components: wrapperComponents,
	}

	if err := transformer.Transform(&wrappedAsset); err != nil {
		return err
	}

	*asset = wrappedAsset

	return nil
}
