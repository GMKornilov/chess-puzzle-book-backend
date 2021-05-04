package api

import (
	"github.com/gin-gonic/gin"
	"github.com/gmkornilov/chess-puzzle-book-backend/internal/dao"
)

type TaskApi struct {
	taskRepository *dao.TaskRepository
}

func NewTaskApi(taskRepo *dao.TaskRepository) *TaskApi {
	return &TaskApi{
		taskRepo,
	}
}

//
func (t *TaskApi) Task(ctx *gin.Context)  {

}
