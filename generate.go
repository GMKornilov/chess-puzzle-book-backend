package chess_puzzle_generator

import (
	"fmt"
	"github.com/freeeve/uci"
	"github.com/notnil/chess"
)

func compareResults(baseRes uci.ScoreResult, cmpRes uci.ScoreResult) bool {
	if baseRes.Mate {
		return cmpRes.Mate && baseRes.Score == cmpRes.Score
	}
	return baseRes.Score - cmpRes.Score <= 50
}

func filterResults(results []uci.ScoreResult) []uci.ScoreResult {
	baseRes := results[0]
	filteredResults := make([]uci.ScoreResult, 0)
	for _, item := range results {
		if compareResults(baseRes, item) {
			filteredResults = append(filteredResults, item)
		}
	}
	return filteredResults
}

func generateCheckmate(game chess.Game, e *uci.Engine, res uci.ScoreResult) (Turn, error) {
	if !res.Mate {
		return Turn{}, fmt.Errorf("given result does not result to mate position")
	}

	beginPos := game.Position()
	firstMove, err := chess.UCINotation{}.Decode(beginPos, res.BestMoves[0])
	if err != nil {
		return Turn{}, err
	}

	if res.Mate && res.Score == 1 {
		resTurn := Turn{
			SanNotation:           chess.AlgebraicNotation{}.Encode(beginPos, firstMove),
			IsLastTurn:            true,
			AnswerTurnSanNotation: "",
			ContinueVariations:    nil,
		}
		return resTurn, err
	}

	game.Move(firstMove)
	ansPos := game.Position()
	ansMove, err := chess.UCINotation{}.Decode(ansPos, res.BestMoves[1])
	if err != nil {
		return Turn{}, err
	}

	fen := game.FEN()
	e.SetFEN(fen)

	results, err := e.GoDepth(res.Score)
	if err != nil {
		return Turn{}, err
	}

	filteredResults := filterResults(results.Results)
	continueTurns := make([]Turn, 0)

	for _, filteredResult := range filteredResults {
		turn, err := generateCheckmate(game, e, filteredResult)
		if err != nil {
			return Turn{}, err
		}
		continueTurns = append(continueTurns, turn)
	}

	resTurn := Turn{
		SanNotation:           chess.AlgebraicNotation{}.Encode(beginPos, firstMove),
		IsLastTurn:            false,
		AnswerTurnSanNotation: chess.AlgebraicNotation{}.Encode(ansPos, ansMove),
		ContinueVariations:    continueTurns,
	}
	return resTurn, nil
}
