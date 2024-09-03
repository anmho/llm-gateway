package llms

type Streamable interface {
	Next()
}
