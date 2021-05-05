package api

import (
	"crypto/md5"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gmkornilov/chess-puzzle-book-backend/internal/dao"
	"github.com/gmkornilov/chess-puzzle-book-backend/internal/scraper"
	"net/http"
	"strconv"
	"sync"
)

type TaskApi struct {
	TaskRepository    dao.TaskRepository
	TaskWorkerFactory *scraper.LichessGameScraperFactory
	activeJobs        map[string]scraper.Worker
	totalJobs         int
	mu                sync.RWMutex
}

func NewTaskApi(taskRepo dao.TaskRepository, taskWorker *scraper.LichessGameScraperFactory) *TaskApi {
	return &TaskApi{
		taskRepo,
		taskWorker,
		make(map[string]scraper.Worker, 0),
		0,
		sync.RWMutex{},
	}
}

func (t *TaskApi) Task(ctx *gin.Context) {
	eloStr := ctx.DefaultQuery("elo", "1500")
	elo, err := strconv.Atoi(eloStr)

	if err != nil {
		ctx.JSON(http.StatusBadRequest, err)
		return
	}

	task, err := t.TaskRepository.GetRandomTaskForElo(elo)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	ctx.JSON(http.StatusOK, task)
}

func (t *TaskApi) StartTask(ctx *gin.Context) {
	name := ctx.Param("username")
	lastStr := ctx.DefaultQuery("last", "20")
	last, err := strconv.Atoi(lastStr)
	if err != nil || last <= 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "last shoul be positive integer",
		})
	}

	worker := t.TaskWorkerFactory.CreateLichessScrapper(name, last)
	t.totalJobs++
	byteValue := []byte(strconv.Itoa(t.totalJobs))
	id := fmt.Sprintf("%x", md5.Sum(byteValue))

	t.mu.Lock()
	defer t.mu.Unlock()
	t.activeJobs[id] = &worker
	worker.StartWork()
	ctx.JSON(http.StatusOK, gin.H{
		"job_id": id,
	})
}

func (t *TaskApi) GetJobStatus(ctx *gin.Context) {
	id := ctx.Param("job_id")
	t.mu.RLock()
	defer t.mu.RUnlock()
	worker, ok := t.activeJobs[id]
	if !ok {
		ctx.AbortWithStatus(http.StatusNotFound)
		return
	}
	done := worker.Done()
	if done {
		delete(t.activeJobs, id)
		if worker.Error() != nil {
			ctx.JSON(http.StatusOK, gin.H{
				"done": done,
				"error": worker.Error().Error(),
			})
		} else {
			ctx.JSON(http.StatusOK, gin.H{
				"done": done,
				"result": worker.Result(),
			})
		}
	} else {
		ctx.JSON(http.StatusOK, gin.H{
			"done": done,
		})
	}
}
