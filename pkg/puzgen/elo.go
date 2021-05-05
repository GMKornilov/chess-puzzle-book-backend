package puzgen

import (
	"github.com/notnil/chess"
	"math"
)

func findMinDepth(turn Turn) int {
	if len(turn.ContinueVariations) == 0 {
		return 1
	}
	var minDepth = -1
	for _, continueTurn := range turn.ContinueVariations {
		if minDepth == -1 {
			minDepth = findMinDepth(continueTurn)
		} else {
			minDepth = int(math.Min(float64(minDepth), float64(findMinDepth(continueTurn))))
		}
	}
	return minDepth + 1
}

func estimatePercent(moves []*chess.Move, game chess.Game, turn Turn) float64 {
	correctMoves := 0
	for _, move := range moves {
		notation := chess.AlgebraicNotation{}.Encode(game.Position(), move)
		var newTurn Turn
		for _, continueTurn := range turn.ContinueVariations {
			if continueTurn.SanNotation == notation {
				newTurn = continueTurn
				break
			}
		}
		if newTurn.SanNotation == "" {
			break
		}
		turn = newTurn
		correctMoves++
	}
	totalMoves := correctMoves + findMinDepth(turn)
	return float64(correctMoves) / float64(totalMoves)
}

func eloCoeff(elo int) int {
	if elo >= 2400 {
		return 10
	}
	if elo >= 2000 {
		return 20
	}
	return 40
}

func estimateElo(moves []*chess.Move, game chess.Game, turn Turn, playerElo int) int {
	// by default we consider task and player elo equal, so expected is always 0.5
	expectedPercent := 0.5

	coeff := eloCoeff(playerElo)

	percent := estimatePercent(moves, game, turn)

	return playerElo + int(float64(coeff) * (percent - expectedPercent))
}

func estimateAllElos(moves []*chess.Move, game chess.Game, turns []Turn, playerElo int) int {
	var maxRes = estimateElo(moves, game, turns[0], playerElo)
	for _, turn := range turns {
		maxRes = int(math.Max(float64(maxRes), float64(estimateElo(moves, game, turn, playerElo))))
	}
	return maxRes
}