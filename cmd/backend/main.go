package main

import (
	"fmt"
	"github.com/gmkornilov/chess-puzzle-generator/pkg/puzgen"
	"github.com/notnil/chess"
	"os"
)

func main() {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	reader, err := os.Open(pwd + "\\cmd\\chesscom.pgn")
	if err != nil {
		panic(err)
	}

	gameFunc, err := chess.PGN(reader)
	if err != nil {
		panic(err)
	}
	game := chess.NewGame(gameFunc)

	tasks, err := puzgen.AnalyzeGame("stockfish", game)

	fmt.Println(len(tasks))

	if err != nil {
		panic(err)
	}

	for _, task := range tasks {
		fmt.Printf("%s\n", task)
	}
}