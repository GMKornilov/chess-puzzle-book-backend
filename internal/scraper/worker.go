package scraper

type Worker interface {
	StartWork()
	Result() interface{}
	Progress() float64
	Done() bool
	Error() error
}
