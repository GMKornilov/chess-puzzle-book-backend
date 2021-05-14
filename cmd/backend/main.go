package main

import (
	"github.com/gin-gonic/gin"
	"github.com/gmkornilov/chess-puzzle-book-backend/internal/api"
	"github.com/gmkornilov/chess-puzzle-book-backend/internal/config"
	"github.com/gmkornilov/chess-puzzle-book-backend/internal/dao"
	"github.com/gmkornilov/chess-puzzle-book-backend/internal/db"
	"github.com/gmkornilov/chess-puzzle-book-backend/internal/scraper"
)

func main() {
	r := gin.Default()
	cfg, err := config.InitBackendConfig()
	if err != nil {
		panic(err)
	}


	db, err := db.NewDbClientBackend(cfg)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	taskRepo := dao.NewTaskRepository(db)

	scrapperFactory := scraper.NewLichessGameScraperFactory(cfg, taskRepo)

	taskApi := api.NewTaskApi(taskRepo, scrapperFactory)

	r.GET("/task", taskApi.Task)
	r.GET("/task/:username", taskApi.StartTask)
	r.GET("/job/:job_id", taskApi.GetJobStatus)

	r.Run(":" + cfg.Server.Port)
}