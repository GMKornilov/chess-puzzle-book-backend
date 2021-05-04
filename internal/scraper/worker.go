package scraper

type Worker interface {
	StartWork()
	Result() interface{}
	Done() bool
	Error() error
}
