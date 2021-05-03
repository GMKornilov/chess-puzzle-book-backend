package puzgen

import (
	"encoding/json"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Task struct {
	StartFEN           string   `json:"start_fen" bson:"start_fen"`
	FirstPossibleTurns []Turn   `json:"first_possible_turns" bson:"first_possible_turns"`
	IsWhiteTurn        bool     `json:"is_white_turn" bson:"is_white_turn"`
	GameData           GameData `json:"game_data" bson:"game_data"`
	TargetELO          int      `json:"target_elo" bson:"target_elo"`
}

type GameData struct {
	WhitePlayer string             `json:"white_player" bson:"white_player"`
	BlackPlayer string             `json:"black_player" bson:"black_player"`
	Date        primitive.DateTime `json:"date" bson:"date"`
}

func (t Task) String() string {
	j, _ := json.MarshalIndent(t, "", "\t")
	return string(j)
}

type Turn struct {
	SanNotation           string `json:"san_notation" bson:"san_notation"`
	IsLastTurn            bool   `json:"is_last_turn,omitempty" bson:"is_last_turn,omitempty"`
	AnswerTurnSanNotation string `json:"answer_turn_san_notation,omitempty" bson:"answer_turn_san_notation,omitempty"`
	ContinueVariations    []Turn `json:"continue_variations,omitempty" bson:"continue_variations,omitempty"`
}

func (t Turn) String() string {
	j, _ := json.MarshalIndent(t, "", "\t")
	return string(j)
}
