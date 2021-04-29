package chess_puzzle_generator

import (
	"github.com/freeeve/uci"
	"github.com/notnil/chess"
	"io"
)

const maxDepth = 10

func AnalyzeGame(path string, r io.Reader) error {
	var e *uci.Engine
	var err error
	if e, err = uci.NewEngine(path); err != nil {
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

func AnalyzeAllGames(path string, r io.Reader) error  {
	var e *uci.Engine
	var err error
	if e, err = uci.NewEngine(path); err != nil {
		return err
	}
	defer e.Close()

	scanner := chess.NewScanner(r)
	for scanner.Scan() {
		game := scanner.Next()
		analyzeGame(game, e)
	}
	return nil
}

func analyzeGame(g *chess.Game, e *uci.Engine) error {
	moves := g.Moves()
	newGame := chess.NewGame()
	for _, move := range moves {
		newGame.Move(move)
	}
	return nil
}

func generateTask(position string, e *uci.Engine) (*Task, error) {
	err := e.SetFEN(position)
	if err != nil {
		return nil, err
	}
	_, err = e.GoDepth(10)
	if err != nil {
		return nil, err
	}

	return nil, nil
}