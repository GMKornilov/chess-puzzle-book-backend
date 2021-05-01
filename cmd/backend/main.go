package main

import (
	"fmt"
	"github.com/gmkornilov/chess-puzzle-generator/pkg/puzgen"
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

	tasks, err := puzgen.AnalyzeGame("stockfish", reader)

	fmt.Println(len(tasks))

	if err != nil {
		panic(err)
	}

	for _, task := range tasks {
		fmt.Printf("%s\n", task)
	}
}