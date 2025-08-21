package builder

type Transformer interface {
	Transform(*Asset) error
}
