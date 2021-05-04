package scraper

type Worker interface {
	StartWork()
	Result() interface{}
	Error() error
}
