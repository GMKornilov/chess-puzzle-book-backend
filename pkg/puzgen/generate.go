package puzgen

import (
	"github.com/freeeve/uci"
	"github.com/notnil/chess"
	"sort"
)

func compareResults(baseRes uci.ScoreResult, cmpRes uci.ScoreResult) bool {
	if baseRes.Mate {
		return cmpRes.Mate && baseRes.Score == cmpRes.Score
	}
	return baseRes.Score-cmpRes.Score <= 50
}

func filterResults(results []uci.ScoreResult) []uci.ScoreResult {
	sort.Slice(results, func(i, j int) bool {
		if results[i].Mate {
			if !results[j].Mate {
				return true
			}
			return results[i].Score <= results[j].Score
		}
		if results[j].Mate {
			return false
		}
		return results[i].Score >= results[j].Score
	})
	baseRes := results[0]
	filteredResults := make([]uci.ScoreResult, 0)
	for _, item := range results {
		if compareResults(baseRes, item) {
			filteredResults = append(filteredResults, item)
		}
	}
	return filteredResults
}

func generateCheckmate(game chess.Game, e *uci.Engine, res uci.ScoreResult, watchedPositions map[string][]Turn) (Turn, error) {
	if !res.Mate {
		return Turn{}, nil
	}

	if res.Score < 0 {
		return Turn{}, nil
	}

	//watchedPositions[game.FEN()] = true

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

	var ansMove *chess.Move
	var ansMoveUci string
	if len(res.BestMoves) == 1 {
		e.SetFEN(game.FEN())
		ansResults, err := e.GoDepth(res.Score, uci.IncludeLowerbounds | uci.IncludeUpperbounds)
		if err != nil {
			return Turn{}, err
		}
		ansMoveUci = ansResults.Results[0].BestMoves[0]
	} else {
		ansMoveUci = res.BestMoves[1]
	}
	ansMove, err = chess.UCINotation{}.Decode(ansPos, ansMoveUci)
	if err != nil {
		return Turn{}, err
	}

	game.Move(ansMove)

	var continueTurns []Turn
	if a, exists := watchedPositions[game.FEN()]; !exists {
		fen := game.FEN()
		e.SetFEN(fen)

		results, err := e.GoDepth(res.Score, uci.IncludeLowerbounds | uci.IncludeUpperbounds)
		if err != nil {
			return Turn{}, err
		}

		filteredResults := filterResults(results.Results)
		continueTurns := make([]Turn, 0)

		for _, filteredResult := range filteredResults {
			turn, err := generateCheckmate(game, e, filteredResult, watchedPositions)
			if err != nil {
				return Turn{}, err
			}
			if turn.SanNotation != "" {
				continueTurns = append(continueTurns, turn)
			}
		}

		watchedPositions[fen] = continueTurns
	} else {
		continueTurns = a
	}


	if len(continueTurns) == 0 {
		return Turn{}, nil
	}
	
	resTurn := Turn{
		SanNotation:           chess.AlgebraicNotation{}.Encode(beginPos, firstMove),
		IsLastTurn:            false,
		AnswerTurnSanNotation: chess.AlgebraicNotation{}.Encode(ansPos, ansMove),
		ContinueVariations:    continueTurns,
	}
	return resTurn, nil
}
