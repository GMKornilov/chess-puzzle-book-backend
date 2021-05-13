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
		log.Println("Req error:" + err.Error())
		return err
	}
	d := json.NewDecoder(resp.Body)

	for {
		var cur LiveMessage
		if !d.More() {
			return l.Main()
		}
		err := d.Decode(&cur)
		if err != nil {
			log.Println("Decode error:" + err.Error())
			return err
		}
		switch cur.Action {
		case "featured":
			var gameStart GameStart
			err = json.Unmarshal(cur.Data, &gameStart)
			if err != nil {
				log.Println("Unmarshal error:" + err.Error())
				return err
			}
			var whiteInd int
			if gameStart.Players[0].Color == "white" {
				whiteInd = 0
			} else {
				whiteInd = 1
			}
			blackInd := 1 - whiteInd
			if l.curAnalyzer != nil {
				close(l.curAnalyzer.GameChan)
			}
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
				Key:   "UTCDate",
				Value: time.Now().Format(puzgen.Layout),
			}
			tm := chess.TagPair{
				Key:   "UTCTime",
				Value: time.Now().Format(puzgen.TimeLayout),
			}

			tags := []chess.TagPair{
				white,
				black,
				whiteElo,
				blackElo,
				date,
				tm,
			}
			log.Printf("New game with start position: %s\n", gameStart.Fen)
			l.curAnalyzer, err = l.NewLiveGameAnalyzer(tags)
			if err != nil {
				log.Println("Error creating analyzer:", err.Error())
				return err
			}
			l.curAnalyzer.StartAnalyze()

		case "fen":
			var gameTurn GameTurn
			err = json.Unmarshal(cur.Data, &gameTurn)
			if err != nil {
				log.Println("Unmarshal error:" + err.Error())
				return err
			}
			gameTurn.Fen += " - - 0 1"
			log.Printf("New position: %s\n", gameTurn.Fen)

			fenFunc, err := chess.FEN(gameTurn.Fen)
			if err != nil {
				log.Println("Unmarshal fen error:" + err.Error())
				return err
			}
			game := chess.NewGame(fenFunc)

			l.curAnalyzer.GameChan <- game
		default:
			return fmt.Errorf("unknown action type from lichess: %s", cur.Action)
		}
	}
}

func (l *LiveLichessScraper) NewLiveGameAnalyzer(tags []chess.TagPair) (*LiveGameAnalyzer, error) {
	e, err := puzgen.SetupEngine(l.stockfishPath, l.stockfishArgs...)
	if err != nil {
		return nil, err
	}
	return &LiveGameAnalyzer{
		tags:     tags,
		engine:   e,
		taskRepo: l.taskRepo,
		// euristic size of chan (we assume we don't put 100 moves while analyzing 1 move)
		GameChan: make(chan *chess.Game, 100),
	}, nil
}

type LiveGameAnalyzer struct {
	tags     []chess.TagPair
	engine   *uci.Engine
	taskRepo dao.TaskRepository
	GameChan chan *chess.Game
}

func (l *LiveGameAnalyzer) StartAnalyze() {
	go l.Analyze()
}

func (l *LiveGameAnalyzer) Analyze() {
	watchedPositions := make(map[string][]puzgen.Turn, 0)
	for game := range l.GameChan {
		for _, tag := range l.tags {
			game.AddTagPair(tag.Key, tag.Value)
		}
		task, err := puzgen.GenerateTaskFromPosition(*game, l.engine, watchedPositions)
		if err != nil {
			log.Println(err.Error())
			return
		}
		if task.StartFEN == "" {
			continue
		}
		log.Printf("Generated task: %+v\n", task)
		err = l.taskRepo.InsertTask(task)
		if err != nil {
			log.Println(err.Error())
			return
		}
	}
}
