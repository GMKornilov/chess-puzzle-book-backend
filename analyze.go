package chess_puzzle_generator

import (
	"github.com/freeeve/uci"
	"github.com/notnil/chess"
	"io"
)

const maxDepth = 10

func setupEngine(path string) (*uci.Engine, error) {
	e, err := uci.NewEngine(path)
	if err != nil {
		return nil, err
	}

	err = e.SetOptions(uci.Options{
		MultiPV: 10,
		Hash:    128,
		Ponder:  false,
		OwnBook: true,
	})
	if err != nil {
		return nil, err
	}
	return e, nil
}


func AnalyzeGame(path string, r io.Reader) error {
	var e *uci.Engine
	var err error
	if e, err = setupEngine(path); err != nil {
		return err
	}
	defer e.Close()

	pgnFunc, err := chess.PGN(r)
	if err != nil {
		return err
	}
	game := chess.NewGame(pgnFunc)
	analyzeGame(game, e)
	return nil
}

func AnalyzeAllGames(path string, r io.Reader) ([]Task, error)  {
	var e *uci.Engine
	var err error
	if e, err = setupEngine(path); err != nil {
		return nil, err
	}
	defer e.Close()

	scanner := chess.NewScanner(r)

	res := make([]Task, 0)

	for scanner.Scan() {
		game := scanner.Next()
		newTasks, err := analyzeGame(game, e)
		if err != nil {
			return nil, err
		}
		res = append(res, newTasks...)
	}
	return res, nil
}

func analyzeGame(g *chess.Game, e *uci.Engine) ([]Task, error) {
	moves := g.Moves()
	newGame := chess.NewGame()
	res := make([]Task, 0)
	for _, move := range moves {
		newGame.Move(move)
		task, err := generateTask(*newGame, e)
		if err != nil {
			return nil, err
		}
		if task.StartFEN != "" {
			res = append(res, task)
		}
	}
	return res, nil
}

func generateTask(game chess.Game, e *uci.Engine) (Task, error) {
	err := e.SetFEN(game.FEN())
	if err != nil {
		return Task{}, err
	}
	result, err := e.GoDepth(10)
	if err != nil {
		return Task{}, err
	}

	filteredResults := filterResults(result.Results)
	possibleTurns := make([]Turn, 0)

	if !result.Results[0].Mate {
		return Task{}, nil
	}

	for _, filteredResult := range filteredResults {
		turn, err := generateCheckmate(game, e, filteredResult)
		if err != nil {
			return Task{}, err
		}
		possibleTurns = append(possibleTurns, turn)
	}

	taskRes := Task{
		StartFEN:           game.FEN(),
		FirstPossibleTurns: possibleTurns,
		IsWhiteTurn:        game.Position().Turn() == chess.Black,
	}
	return taskRes, nil
}