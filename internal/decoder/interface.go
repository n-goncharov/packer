package decoder

type Interface interface {
	Decode(encoding Encoding) (FitnessValue, error)
}
