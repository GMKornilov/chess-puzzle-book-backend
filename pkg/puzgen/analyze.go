package puzgen

import (
	"github.com/freeeve/uci"
	"github.com/notnil/chess"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

const (
	maxDepth = 10
	layout = "2006.01.02"
)

func setupEngine(path string, arg ...string) (*uci.Engine, error) {
	e, err := uci.NewEngine(path, arg...)
	if err != nil {
		return nil, err
	}

	err = e.SetOptions(uci.Options{
		MultiPV: maxDepth,
		Hash:    128,
		Ponder:  false,
		OwnBook: true,
	})
	if err != nil {
		return nil, err
	}
	return e, nil
}


func AnalyzeGame(path string, game *chess.Game, arg ...string) ([]Task, error) {
	var e *uci.Engine
	var err error
	if e, err = setupEngine(path, arg...); err != nil {
		return nil, err
	}
	defer e.Close()
	tasks, err := analyzeGame(game, e)
	if err != nil {
		return nil, err
	}
	gameTime, err := time.Parse(layout, game.GetTagPair("Date").Value)
	if err != nil {
		return nil, err
	}
	for ind := range tasks {
		tasks[ind].GameData = GameData{
			WhitePlayer: game.GetTagPair("White").Value,
			BlackPlayer: game.GetTagPair("Black").Value,
			Date:        primitive.NewDateTimeFromTime(gameTime),
		}
	}
	return tasks, nil
}

func AnalyzeAllGames(path string, games []*chess.Game, arg ...string) ([]Task, error)  {
	var e *uci.Engine
	var err error
	if e, err = setupEngine(path, arg...); err != nil {
		return nil, err
	}
	defer e.Close()

	res := make([]Task, 0)

	for _, game := range games {
		newTasks, err := analyzeGame(game, e)
		if err != nil {
			return nil, err
		}
		gameTime, err := time.Parse(layout, game.GetTagPair("Date").Value)
		if err != nil {
			return nil, err
		}
		for ind := range newTasks {
			newTasks[ind].GameData = GameData{
				WhitePlayer: game.GetTagPair("White").Value,
				BlackPlayer: game.GetTagPair("Black").Value,
				Date:        primitive.NewDateTimeFromTime(gameTime),
			}
		}
		res = append(res, newTasks...)
	}
	return res, nil
}

func analyzeGame(g *chess.Game, e *uci.Engine) ([]Task, error) {
	watchedPositions := make(map[string]bool, 0)
	moves := g.Moves()
	newGame := chess.NewGame()
	res := make([]Task, 0)
	for _, move := range moves {
		newGame.Move(move)
		task, err := GenerateTaskFromPosition(*newGame, e, watchedPositions)
		if err != nil {
			return nil, err
		}
		if task.StartFEN != "" {
			res = append(res, task)
		}
	}
	return res, nil
}

func GenerateTaskFromPosition(game chess.Game, e *uci.Engine, watchedPositions map[string]bool) (Task, error) {
	if _, ok := watchedPositions[game.FEN()]; ok {
		return Task{}, nil
	}

	err := e.SetFEN(game.FEN())
	if err != nil {
		return Task{}, err
	}
	result, err := e.GoDepth(maxDepth)
	if err != nil {
		return Task{}, err
	}

	if len(result.Results) == 0 {
		return Task{}, nil
	}
	filteredResults := filterResults(result.Results)
	possibleTurns := make([]Turn, 0)

	if !result.Results[0].Mate || result.Results[0].Score < 1 {
		return Task{}, nil
	}

	for _, filteredResult := range filteredResults {
		turn, err := generateCheckmate(game, e, filteredResult, watchedPositions)
		if err != nil {
			return Task{}, err
		}
		if turn.SanNotation != "" {
			possibleTurns = append(possibleTurns, turn)
		}
	}

	taskRes := Task{
		StartFEN:           game.FEN(),
		FirstPossibleTurns: possibleTurns,
		IsWhiteTurn:        game.Position().Turn() == chess.White,
	}
	return taskRes, nil
}