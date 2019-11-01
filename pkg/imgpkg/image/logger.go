package image

type Logger interface {
	BeginLinef(pattern string, args ...interface{})
}
