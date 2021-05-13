package puzgen

import (
	"github.com/freeeve/uci"
	"github.com/notnil/chess"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"strconv"
	"time"
)

const (
	maxDepth   = 6
	Layout     = "2006.01.02"
	TimeLayout = "15:04:05"
)

func SetupEngine(path string, arg ...string) (*uci.Engine, error) {
	e, err := uci.NewEngine(path, arg...)
	if err != nil {
		return nil, err
	}

	err = e.SetOptions(uci.Options{
		MultiPV: maxDepth,
		Hash:    128,
		Ponder:  false,
		OwnBook: true,
	})
	if err != nil {
		return nil, err
	}
	return e, nil
}

func AnalyzeGame(path string, game *chess.Game, arg ...string) ([]Task, error) {
	var e *uci.Engine
	var err error
	if e, err = SetupEngine(path, arg...); err != nil {
		return nil, err
	}
	defer e.Close()
	tasks, err := analyzeGame(game, e)
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

func AnalyzeAllGames(path string, games []*chess.Game, progressChan chan<- struct{}, arg ...string) ([]Task, error) {
	var e *uci.Engine
	var err error
	if e, err = SetupEngine(path, arg...); err != nil {
		return nil, err
	}
	defer e.Close()

	res := make([]Task, 0)

	for _, game := range games {
		newTasks, err := analyzeGame(game, e)
		if err != nil {
			return nil, err
		}
		progressChan <- struct{}{}
		res = append(res, newTasks...)
	}
	return res, nil
}

func analyzeGame(g *chess.Game, e *uci.Engine) ([]Task, error) {
	watchedPositions := make(map[string][]Turn, 0)
	moves := g.Moves()
	newGame := chess.NewGame()
	for _, tagPair := range g.TagPairs() {
		newGame.AddTagPair(tagPair.Key, tagPair.Value)
	}
	res := make([]Task, 0)
	for ind, move := range moves {
		newGame.Move(move)
		task, err := GenerateTaskFromPosition(*newGame, e, watchedPositions)
		if err != nil {
			return nil, err
		}
		if task.StartFEN != "" {
			var eloStr string
			if g.Position().Turn() == chess.White {
				eloStr = g.GetTagPair("WhiteElo").Value
			} else {
				eloStr = g.GetTagPair("BlackElo").Value
			}
			elo, _ := strconv.Atoi(eloStr)
			task.TargetELO = estimateAllElos(moves[ind:], *g, task.FirstPossibleTurns, elo)
			res = append(res, task)
		}
	}
	return res, nil
}

func GenerateTaskFromPosition(game chess.Game, e *uci.Engine, watchedPositions map[string][]Turn) (Task, error) {
	if _, ok := watchedPositions[game.FEN()]; ok {
		return Task{}, nil
	}

	err := e.SetFEN(game.FEN())
	if err != nil {
		return Task{}, err
	}
	result, err := e.GoDepth(maxDepth, uci.IncludeLowerbounds | uci.IncludeUpperbounds)
	if err != nil {
		return Task{}, err
	}

	if len(result.Results) == 0 {
		return Task{}, nil
	}
	filteredResults := filterResults(result.Results)
	possibleTurns := make([]Turn, 0)

	if !result.Results[0].Mate || result.Results[0].Score <= 1 {
		return Task{}, nil
	}

	for _, filteredResult := range filteredResults {
		turn, err := generateCheckmate(game, e, filteredResult, watchedPositions)
		if err != nil {
			return Task{}, err
		}
		if turn.SanNotation != "" {
			possibleTurns = append(possibleTurns, turn)
		}
	}

	var eloStr string
	if game.Position().Turn() == chess.White {
		eloStr = game.GetTagPair("WhiteElo").Value
	} else {
		eloStr = game.GetTagPair("BlackElo").Value
	}
	elo, _ := strconv.Atoi(eloStr)

	extraTime, err := time.Parse(TimeLayout, game.GetTagPair("UTCTime").Value)
	if err != nil {
		return Task{}, err
	}

	gameTime, err := time.Parse(Layout, game.GetTagPair("UTCDate").Value)
	if err != nil {
		return Task{}, err
	}

	gameTime = gameTime.Add(time.Second * time.Duration(extraTime.Second()) +
		time.Minute * time.Duration(extraTime.Minute()) +
		time.Hour * time.Duration(extraTime.Hour()))

	if len(possibleTurns) == 0 {
		return Task{}, nil
	}

	taskRes := Task{
		StartFEN:           game.FEN(),
		FirstPossibleTurns: possibleTurns,
		IsWhiteTurn:        game.Position().Turn() == chess.White,
		TargetELO:          elo,
		GameData: GameData{
			WhitePlayer: game.GetTagPair("White").Value,
			BlackPlayer: game.GetTagPair("Black").Value,
			Date:        primitive.NewDateTimeFromTime(gameTime),
		},
	}

	return taskRes, nil
}
