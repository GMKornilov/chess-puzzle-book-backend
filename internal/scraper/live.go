package scraper

import (
	"encoding/json"
	"fmt"
	"github.com/freeeve/uci"
	"github.com/gmkornilov/chess-puzzle-book-backend/internal/config"
	"github.com/gmkornilov/chess-puzzle-book-backend/internal/dao"
	"github.com/gmkornilov/chess-puzzle-book-backend/pkg/puzgen"
	"github.com/notnil/chess"
	"log"
	"net/http"
	"strconv"
	"time"
)

type LiveLichessScraper struct {
	taskRepo      dao.TaskRepository
	curAnalyzer   *LiveGameAnalyzer
	stockfishPath string
	stockfishArgs []string
}

func NewLiveLichessScraper(repository dao.TaskRepository, configuration config.ScraperConfiguration) *LiveLichessScraper {
	return &LiveLichessScraper{
		taskRepo:      repository,
		curAnalyzer:   nil,
		stockfishPath: configuration.Stockfish.Path,
		stockfishArgs: configuration.Stockfish.Args,
	}
}

func (l *LiveLichessScraper) Main() error {
	resp, err := http.Get("https://lichess.org/api/tv/feed")
	if err != nil {
		return err
	}
	d := json.NewDecoder(resp.Body)

	for {
		var cur LiveMessage
		err := d.Decode(&cur)
		if err != nil {
			return err
		}
		switch cur.Action {
		case "featured":
			var gameStart GameStart
			err = json.Unmarshal(cur.Data, &gameStart)
			if err != nil {
				return err
			}
			var whiteInd int
			if gameStart.Players[0].Color == "white" {
				whiteInd = 0
			} else {
				whiteInd = 1
			}
			blackInd := 1 - whiteInd

			close(l.curAnalyzer.MoveChan)
			white := chess.TagPair{
				Key:   "White",
				Value: gameStart.Players[whiteInd].User.Name,
			}
			black := chess.TagPair{
				Key:   "Black",
				Value: gameStart.Players[blackInd].User.Name,
			}
			whiteElo := chess.TagPair{
				Key:   "WhiteElo",
				Value: strconv.Itoa(gameStart.Players[whiteInd].Rating),
			}
			blackElo := chess.TagPair{
				Key:   "BlackElo",
				Value: strconv.Itoa(gameStart.Players[blackInd].Rating),
			}
			date := chess.TagPair{
				Key:   "Date",
				Value: time.Now().Format(puzgen.Layout),
			}

			tags := []chess.TagPair{
				white,
				black,
				whiteElo,
				blackElo,
				date,
			}
			log.Printf("New pos: %s\n", gameStart.Fen)
			l.curAnalyzer, err = l.NewLiveGameAnalyzer(tags, gameStart.Fen)
			if err != nil {
				return err
			}
			l.curAnalyzer.StartAnalyze()

		case "fen":
			var gameTurn GameTurn
			err = json.Unmarshal(cur.Data, &gameTurn)
			if err != nil {
				return err
			}
			log.Printf("New move: %s\n", gameTurn.TurnUciNotation)
			l.curAnalyzer.MoveChan <- gameTurn.TurnUciNotation
		default:
			return fmt.Errorf("unknown action type from lichess: %s", cur.Action)
		}
	}
}

func (l *LiveLichessScraper) NewLiveGameAnalyzer(tags []chess.TagPair, fen string) (*LiveGameAnalyzer, error) {
	e, err := puzgen.SetupEngine(l.stockfishPath, l.stockfishArgs...)
	if err != nil {
		return nil, err
	}

	fenFunc, err := chess.FEN(fen)
	if err != nil {
		return nil, err
	}
	game := chess.NewGame(chess.UseNotation(chess.UCINotation{}), fenFunc)
	for _, tag := range tags {
		game.AddTagPair(tag.Key, tag.Value)
	}
	return &LiveGameAnalyzer{
		engine:   e,
		game:     game,
		taskRepo: l.taskRepo,
		// euristic size of chan (we assume we don't put 100 moves while analyzing 1 move)
		MoveChan: make(chan string, 100),
	}, nil
}

type LiveGameAnalyzer struct {
	engine   *uci.Engine
	game     *chess.Game
	taskRepo dao.TaskRepository
	MoveChan chan string
}

func (l *LiveGameAnalyzer) StartAnalyze() {
	go l.Analyze()
}

func (l *LiveGameAnalyzer) Analyze() {
	watchedPositions := make(map[string] bool, 0)
	for moveUci := range l.MoveChan {
		err := l.game.MoveStr(moveUci)
		if err != nil {
			log.Println(err.Error())
			return
		}
		task, err := puzgen.GenerateTaskFromPosition(*l.game, l.engine, watchedPositions)
		if err != nil {
			log.Println(err.Error())
			return
		}
		if task.StartFEN == "" {
			continue
		}
		err = l.taskRepo.InsertTask(task)
		if err != nil {
			log.Println(err.Error())
			return
		}
	}
}
