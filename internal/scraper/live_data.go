package scraper

import "encoding/json"

type PlayerInfo struct {
	Color string `json:"color"`
	User  struct {
		Name  string `json:"name"`
		Id    string `json:"id"`
		Title string `json:"title"`
	}
	Rating int `json:"rating"`
}

type GameStart struct {
	Id          string       `json:"id"`
	Orientation string       `json:"orientation"`
	Players     []PlayerInfo `json:"players"`
	Fen         string       `json:"fen"`
}

type GameTurn struct {
	Fen             string `json:"fen"`
	TurnUciNotation string `json:"lm"`
	WhiteSomeDummy  int    `json:"wc"`
	BlackSomeDummy  int    `json:"bc"`
}

type LiveMessage struct {
	Action string          `json:"t"`
	Data   json.RawMessage `json:"d"`
}
