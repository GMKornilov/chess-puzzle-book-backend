package main

import (
	"github.com/gmkornilov/chess-puzzle-book-backend/internal/config"
	"github.com/gmkornilov/chess-puzzle-book-backend/internal/dao"
	"github.com/gmkornilov/chess-puzzle-book-backend/internal/db"
	"github.com/gmkornilov/chess-puzzle-book-backend/internal/scraper"
)

func main() {
	cfg, err := config.InitScraperConfig()
	if err != nil {
		panic(err)
	}

	db, err := db.NewDbClientScraper(cfg)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	taskRepo := dao.NewTaskRepository(db)
	analyzer := scraper.NewLiveLichessScraper(taskRepo, *cfg)

	err = analyzer.Main()
	if err != nil {
		panic(err)
	}
}
