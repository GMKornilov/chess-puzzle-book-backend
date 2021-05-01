package puzgen

import "encoding/json"

type Task struct {
	StartFEN           string `json:"start_fen"`
	FirstPossibleTurns []Turn `json:"first_possible_turns"`
	IsWhiteTurn        bool   `json:"is_white_turn"`
}

func (t Task) String() string {
	j, _ := json.MarshalIndent(t, "", "\t")
	return string(j)
}

type Turn struct {
	SanNotation           string `json:"san_notation"`
	IsLastTurn            bool   `json:"is_last_turn,omitempty"`
	AnswerTurnSanNotation string `json:"answer_turn_san_notation,omitempty"`
	ContinueVariations    []Turn `json:"continue_variations,omitempty"`
}

func (t Turn) String() string {
	j, _ := json.MarshalIndent(t, "", "\t")
	return string(j)
}
