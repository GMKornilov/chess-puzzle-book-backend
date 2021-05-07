package scraper

import (
	"fmt"
	"github.com/gmkornilov/chess-puzzle-book-backend/internal/config"
	"github.com/gmkornilov/chess-puzzle-book-backend/internal/dao"
	"github.com/gmkornilov/chess-puzzle-book-backend/pkg/puzgen"
	"github.com/notnil/chess"
	"log"
	"net/http"
	"sync"
)

type LichessGameScraperFactory struct {
	StockfishPath string
	StockfishArgs []string
	TaskRepo      dao.TaskRepository
}

func NewLichessGameScraperFactory(cfg *config.BackendConfiguration, taskRepo dao.TaskRepository) *LichessGameScraperFactory {
	return &LichessGameScraperFactory{
		StockfishPath: cfg.Stockfish.Path,
		StockfishArgs: cfg.Stockfish.Args,
		TaskRepo:      taskRepo,
	}
}

func (f LichessGameScraperFactory) CreateLichessScrapper(nickname string, last int) LichessGameScraper {
	return LichessGameScraper{
		nickname:      nickname,
		last:          last,
		stockfishPath: f.StockfishPath,
		stockfishArgs: f.StockfishArgs,
		taskRepo:      f.TaskRepo,
		done:          false,
	}
}

type LichessGameScraper struct {
	mu    sync.Mutex
	tasks []puzgen.Task
	err   error
	done  bool

	loadedTasks  bool
	overallTasks int
	doneTasks    int

	nickname string
	last     int

	taskRepo      dao.TaskRepository
	stockfishPath string
	stockfishArgs []string
}

func (l *LichessGameScraper) Done() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.done
}

func (l *LichessGameScraper) Progress() float64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.done {
		return 1
	}
	if !l.loadedTasks {
		return 0
	}
	return 0.1 + 0.9*float64(l.doneTasks)/float64(l.overallTasks)
}

func (l *LichessGameScraper) StartWork() {
	go l.Scrap()
}

func (l *LichessGameScraper) Result() interface{} {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.tasks
}

func (l *LichessGameScraper) Error() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.err
}

func (l *LichessGameScraper) Scrap() {
	lastTask, err := l.taskRepo.GetLastUserTask(l.nickname)
	if err != nil {
		l.mu.Lock()
		defer l.mu.Unlock()
		log.Println(err.Error())
		l.err = fmt.Errorf("error fetching last %s game", l.nickname)
		l.done = true
		return
	}

	url := fmt.Sprintf("https://lichess.org/api/games/user/%s?max=%d", l.nickname, l.last)
	if lastTask.StartFEN != "" {
		// add one second offset
		url += fmt.Sprintf("&since=%d", lastTask.GameData.Date+1000)
	}
	log.Println(url)

	games, err := l.GetGamesByUrl(url)
	if err != nil {
		l.mu.Lock()
		defer l.mu.Unlock()
		log.Println(err.Error())
		l.done = true
		if _, ok := err.(userNotFound); ok {
			l.err = fmt.Errorf("user %s doesn't exist on lichess", l.nickname)
		} else {
			l.err = fmt.Errorf("error fetching %s games", l.nickname)
		}
		return
	}

	var gamesInDb = 0
	var doneTasks []puzgen.Task
	if lastTask.StartFEN == "" || l.last-len(games) == 0 {
		doneTasks = []puzgen.Task{}
	} else {
		doneTasks, gamesInDb, err = l.taskRepo.GetLastUserTasks(l.nickname, int64(l.last-len(games)))
		if err != nil {
			l.mu.Lock()
			defer l.mu.Unlock()
			l.done = true
			log.Println(err.Error())
			l.err = fmt.Errorf("error getting already parsed tasks")
			return
		}
	}

	if len(games)+gamesInDb <= l.last {
		firstTask, err := l.taskRepo.GetFirstUserTask(l.nickname)
		if err != nil {
			l.mu.Lock()
			defer l.mu.Unlock()
			log.Println(err.Error())
			l.err = fmt.Errorf("error fetching first %s game", l.nickname)
			l.done = true
			return
		}
		if firstTask.GameData != (puzgen.Task{}).GameData {
			url = fmt.Sprintf("https://lichess.org/api/games/user/%s?max=%d&until=%d", l.nickname, l.last-(gamesInDb+len(games)), firstTask.GameData.Date-1000)
			log.Println(url)
			gamesBefore, err := l.GetGamesByUrl(url)
			if err != nil {
				l.mu.Lock()
				defer l.mu.Unlock()
				log.Println(err.Error())
				l.done = true
				if _, ok := err.(userNotFound); ok {
					l.err = fmt.Errorf("user %s doesn't exist on lichess", l.nickname)
				} else {
					l.err = fmt.Errorf("error fetching %s games", l.nickname)
				}
				return
			}
			games = append(games, gamesBefore...)
		}
	}

	l.mu.Lock()
	l.loadedTasks = true
	l.overallTasks = l.last
	l.doneTasks = gamesInDb
	l.mu.Unlock()

	progressChan := make(chan struct{}, l.overallTasks)
	go func(l *LichessGameScraper, progressChan <-chan struct{}) {
		for range progressChan {
			l.mu.Lock()
			l.doneTasks++
			l.mu.Unlock()
		}
	}(l, progressChan)

	tasks, err := puzgen.AnalyzeAllGames(l.stockfishPath, games, progressChan, l.stockfishArgs...)
	close(progressChan)
	if err != nil {
		l.mu.Lock()
		defer l.mu.Unlock()
		log.Println(err)
		l.err = fmt.Errorf("error generating puzzles")
		l.done = true
		return
	}

	err = l.taskRepo.InsertAllTasks(tasks)
	if err != nil {
		l.mu.Lock()
		defer l.mu.Unlock()
		l.err = fmt.Errorf("error saving tasks to db")
		l.done = true
		return
	}

	tasks = append(tasks, doneTasks...)
	l.mu.Lock()
	defer l.mu.Unlock()
	l.tasks = tasks
	l.done = true
}

func (l *LichessGameScraper) GetGamesByUrl(url string) ([]*chess.Game, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, userNotFound{fmt.Errorf("user %s doesn't exist on lichess", l.nickname)}
	}

	scanner := chess.NewScanner(resp.Body)
	buggedGames := make([]*chess.Game, 0)
	for scanner.Scan() {
		buggedGames = append(buggedGames, scanner.Next())
	}
	games := make([]*chess.Game, 0)
	var tagGame *chess.Game
	for i, game := range buggedGames {
		if i%3 == 0 {
			games = append(games, game)
		} else if i%3 == 1 {
			tagGame = game
		} else {
			for _, tagPair := range tagGame.TagPairs() {
				game.AddTagPair(tagPair.Key, tagPair.Value)
			}
			games = append(games, game)
		}
	}
	return games, nil
}

type userNotFound struct {
	error
}
