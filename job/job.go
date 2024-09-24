package job

import (
	"bytes"
	"obsidian-web/config"
	"obsidian-web/logger"
	"obsidian-web/noteloader"
	"os/exec"

	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
)

func Start() {
	c := cron.New()
	_, err := c.AddFunc("12 4 * * *", func() {
		logger.Info("开始执行git pull")
		cmd := exec.Command("git", "pull")
		cmd.Dir = config.Get().NotePath
		sout := bytes.NewBuffer(nil)
		cmd.Stdout = sout
		serr := bytes.NewBuffer(nil)
		cmd.Stderr = serr
		err := cmd.Run()
		if err != nil {
			logger.Error(errors.WithStack(err))
			return
		}

		// 需要重新解析文件
		noteloader.Load()
	})
	if err != nil {
		logger.Fatal(errors.WithStack(err))
	} else {
		c.Start()
	}
}
