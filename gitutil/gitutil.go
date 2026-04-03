package gitutil

import (
	"fmt"
	"obsidian-web/config"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing/transport"
	"github.com/go-git/go-git/v6/plumbing/transport/http"
	"github.com/pkg/errors"
)

// Result 记录 pull 操作的输出信息
type Result struct {
	Changed bool   // 是否有文件变更
	Msg     string // 操作描述
}

// Pull 执行 git pull，如果本地仓库不存在则先 clone
// 仅在配置了 git_url 时生效，否则返回错误提示
func Pull() (*Result, error) {
	cfg := config.Get()
	if cfg.GitURL == "" {
		return nil, errors.New("未配置 git_url，请在 config.yml 中设置")
	}

	var auth transport.AuthMethod
	if cfg.GitToken != "" {
		auth = &http.BasicAuth{Username: "token", Password: cfg.GitToken}
	}

	// 尝试打开已有仓库
	repo, err := git.PlainOpen(cfg.NotePath)
	if err != nil {
		// 仓库不存在，执行 clone
		_, err = git.PlainClone(cfg.NotePath, &git.CloneOptions{
			URL:  cfg.GitURL,
			Auth: auth,
		})
		if err != nil {
			return nil, errors.Wrap(err, "clone 失败")
		}
		return &Result{
			Changed: true,
			Msg:     "clone 成功",
		}, nil
	}

	// 执行 pull
	wt, err := repo.Worktree()
	if err != nil {
		return nil, errors.Wrap(err, "获取 worktree 失败")
	}

	err = wt.Pull(&git.PullOptions{
		Auth: auth,
	})
	if err != nil {
		if err == git.NoErrAlreadyUpToDate {
			return &Result{
				Changed: false,
				Msg:     "已是最新",
			}, nil
		}
		return nil, errors.Wrap(err, "pull 失败")
	}

	return &Result{
		Changed: true,
		Msg:     "pull 成功",
	}, nil
}

// IsConfigured 检查是否配置了 go-git 拉取
func IsConfigured() bool {
	return config.Get().GitURL != ""
}

// StatusText 返回当前配置状态描述
func StatusText() string {
	cfg := config.Get()
	if cfg.GitURL == "" {
		return "未配置 git_url，使用 go-git 内置模式不可用"
	}
	tokenStatus := "已配置"
	if cfg.GitToken == "" {
		tokenStatus = "未配置（公开仓库可正常使用）"
	}
	return fmt.Sprintf("仓库: %s, Token: %s", cfg.GitURL, tokenStatus)
}
