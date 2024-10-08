package handles

import (
	"context"

	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/internal/search"
	"github.com/alist-org/alist/v3/internal/setting"
	"github.com/alist-org/alist/v3/server/common"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type UpdateIndexReq struct {
	Paths    []string `json:"paths"`
	MaxDepth int      `json:"max_depth"`
	//IgnorePaths []string `json:"ignore_paths"`
}

func BuildIndex(c *gin.Context) {
	if search.Running() {
		common.ErrorStrResp(c, "请先停止索引运行", 400)
		return
	}
	go func() {
		ctx := context.Background()
		err := search.Clear(ctx)
		if err != nil {
			log.Errorf("clear index error: %+v", err)
			return
		}
		err = search.BuildIndex(context.Background(), []string{"/"},
			conf.SlicesMap[conf.IgnorePaths], setting.GetInt(conf.MaxIndexDepth, 20), true)
		if err != nil {
			log.Errorf("build index error: %+v", err)
		}
	}()
	common.SuccessStrResp(c, "索引构建成功")
}

func UpdateIndex(c *gin.Context) {
	var req UpdateIndexReq
	if err := c.ShouldBind(&req); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	if search.Running() {
		common.ErrorStrResp(c, "请先停止索引运行", 400)
		return
	}
	if !search.Config(c).AutoUpdate {
		common.ErrorStrResp(c, "当前索引不支持更新", 400)
		return
	}
	go func() {
		ctx := context.Background()
		for _, path := range req.Paths {
			err := search.Del(ctx, path)
			if err != nil {
				log.Errorf("delete index on %s error: %+v", path, err)
				return
			}
		}
		err := search.BuildIndex(context.Background(), req.Paths,
			conf.SlicesMap[conf.IgnorePaths], req.MaxDepth, false)
		if err != nil {
			log.Errorf("update index error: %+v", err)
		}
	}()
	common.SuccessStrResp(c, "索引更新成功")
}

func StopIndex(c *gin.Context) {
	quit := search.Quit.Load()
	if quit == nil {
		common.ErrorStrResp(c, "当前索引未运行", 400)
		return
	}
	select {
	case *quit <- struct{}{}:
	default:
	}
	common.SuccessResp(c, "索引停止成功")
}

func ClearIndex(c *gin.Context) {
	if search.Running() {
		common.ErrorStrResp(c, "请先停止索引运行", 400)
		return
	}
	search.Clear(c)
	search.WriteProgress(&model.IndexProgress{
		ObjCount:     0,
		IsDone:       true,
		LastDoneTime: nil,
		Error:        "",
	})
	common.SuccessStrResp(c, "索引清除成功")
}

func GetProgress(c *gin.Context) {
	progress, err := search.Progress()
	if err != nil {
		common.ErrorResp(c, err, 500)
		return
	}
	common.SuccessResp(c, progress)
}
