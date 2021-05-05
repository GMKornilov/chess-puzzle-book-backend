package scraper

import (
	"fmt"
	"github.com/gmkornilov/chess-puzzle-book-backend/internal/config"
	"github.com/gmkornilov/chess-puzzle-book-backend/internal/dao"
	"github.com/gmkornilov/chess-puzzle-book-backend/pkg/puzgen"
	"github.com/notnil/chess"
	"net/http"
	"sync"
	"time"
)

type LichessGameScraperFactory struct {
	StockfishPath string
	StockfishArgs []string
	TaskRepo      dao.TaskRepository
}

func NewLichessGameScraperFactory(cfg *config.Configuration, taskRepo dao.TaskRepository) *LichessGameScraperFactory {
	return &LichessGameScraperFactory{
		StockfishPath: cfg.Stockfish.Path,
		StockfishArgs: cfg.Stockfish.Args,
		TaskRepo:      taskRepo,
	}
}

func (f LichessGameScraperFactory) CreateLichessScrapper(nickname string) LichessGameScraper {
	return LichessGameScraper{
		nickname:      nickname,
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

	taskRepo      dao.TaskRepository
	nickname      string
	stockfishPath string
	stockfishArgs []string
}

func (l *LichessGameScraper) Done() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.done
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
	url := fmt.Sprintf("https://lichess.org/api/games/user/%s?since=%d", l.nickname, time.Now().AddDate(0, -1, 0).Unix())
	fmt.Println(url)
	resp, err := http.Get(url)
	if err != nil {
		l.mu.Lock()
		defer l.mu.Unlock()
		l.err = fmt.Errorf("error fetching %s games", l.nickname)
		l.done = true
		return
	}

	if resp.StatusCode == http.StatusNotFound {
		l.mu.Lock()
		defer l.mu.Unlock()
		l.err = fmt.Errorf("user %s doesn't exist on lichess", l.nickname)
		return
	}

	defer resp.Body.Close()
	scanner := chess.NewScanner(resp.Body)
	buggedGames := make([]*chess.Game, 0)
	for scanner.Scan() {
		buggedGames = append(buggedGames, scanner.Next())
	}
	games := make([]*chess.Game, 0)
	var tagGame *chess.Game
	for i, game := range buggedGames {
		if i % 3 == 0 {
			games = append(games, game)
		} else if i % 3 == 1 {
			tagGame = game
		} else {
			for _, tagPair := range tagGame.TagPairs() {
				game.AddTagPair(tagPair.Key, tagPair.Value)
			}
			games = append(games, game)
		}
	}

	tasks, err := puzgen.AnalyzeAllGames(l.stockfishPath, games, l.stockfishArgs...)
	if err != nil {
		l.mu.Lock()
		defer l.mu.Unlock()
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

	l.mu.Lock()
	defer l.mu.Unlock()
	l.tasks = tasks
	l.done = true
}
