package sitetools

type Transformer interface {
	Transform(*Asset) error
}
