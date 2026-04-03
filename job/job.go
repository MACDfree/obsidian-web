package job

import (
	"obsidian-web/gitutil"
	"obsidian-web/logger"
	"obsidian-web/noteloader"

	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
)

func Start() {
	c := cron.New()
	_, err := c.AddFunc("12 4 * * *", func() {
		logger.Info("开始执行git pull")

		result, err := gitutil.Pull()
		if err != nil {
			logger.Error(errors.WithStack(err))
			return
		}

		if result.Changed {
			noteloader.Load()
		}
		logger.Infof("git pull 完成: %s", result.Msg)
	})
	if err != nil {
		logger.Fatal(errors.WithStack(err))
	} else {
		c.Start()
	}
}
