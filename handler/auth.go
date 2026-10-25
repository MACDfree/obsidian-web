package handler

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"obsidian-web/config"
	"obsidian-web/logger"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

const (
	loginFailWindow = 10 * time.Minute
	loginFailLimit  = 5
)

type loginFailRecord struct {
	count    int
	windowAt time.Time
}

var (
	loginFailMu   sync.Mutex
	loginFailures = make(map[string]*loginFailRecord)
)

func renderAuthError(ctx *gin.Context, status int, msg string) {
	ctx.HTML(status, "auth.html", gin.H{
		"Auth": gin.H{
			"IsLogin": false,
		},
		"Site": gin.H{
			"Title": config.Get().Title,
		},
		"CurrentPath": "/auth",
		"Error":       msg,
	})
}

func AuthPage(ctx *gin.Context) {
	isLogin := ctx.MustGet("isLogin").(bool)
	if isLogin {
		ctx.Redirect(302, "/")
		return
	}

	if config.Get().Password == "" {
		renderAuthError(ctx, http.StatusOK, "站点未配置访问密码，暂时无法登录。请在 config.yml 中设置 password。")
		return
	}

	ctx.HTML(http.StatusOK, "auth.html", gin.H{
		"Auth": gin.H{
			"IsLogin": isLogin,
		},
		"Site": gin.H{
			"Title": config.Get().Title,
		},
		"CurrentPath": "/auth",
	})
}

func Auth(ctx *gin.Context) {
	isLogin := ctx.MustGet("isLogin").(bool)
	if isLogin {
		ctx.Redirect(302, "/")
		return
	}

	// 未配置密码时禁止登录，避免空密码直接放行
	expected := config.Get().Password
	if expected == "" {
		renderAuthError(ctx, http.StatusForbidden, "站点未配置访问密码，暂时无法登录。请在 config.yml 中设置 password。")
		return
	}

	ip := ctx.ClientIP()
	if loginLocked(ip) {
		logger.Warnf("登录尝试次数超限 ip=%s", ip)
		renderAuthError(ctx, http.StatusTooManyRequests, "尝试次数过多，请稍后再试。")
		return
	}

	if !passwordMatches(ctx.PostForm("password"), expected) {
		recordLoginFailure(ip)
		logger.Warnf("登录失败 ip=%s", ip)
		renderAuthError(ctx, http.StatusUnauthorized, "密码错误。")
		return
	}

	clearLoginFailures(ip)

	session := sessions.Default(ctx)
	session.Options(sessions.Options{
		Path:     "/",
		MaxAge:   3600,
		HttpOnly: true,
		Secure:   isHTTPS(ctx),
		SameSite: http.SameSiteLaxMode,
	})
	session.Set("isLogin", true)
	if err := session.Save(); err != nil {
		logger.Error(errors.WithStack(err))
		renderAuthError(ctx, http.StatusInternalServerError, "登录失败，请稍后再试。")
		return
	}

	ctx.Redirect(302, "/")
}

func isHTTPS(ctx *gin.Context) bool {
	return ctx.Request.TLS != nil || strings.EqualFold(ctx.GetHeader("X-Forwarded-Proto"), "https")
}

func passwordMatches(input, expected string) bool {
	inputHash := sha256.Sum256([]byte(input))
	expectedHash := sha256.Sum256([]byte(expected))
	return subtle.ConstantTimeCompare(inputHash[:], expectedHash[:]) == 1
}

func loginLocked(ip string) bool {
	loginFailMu.Lock()
	defer loginFailMu.Unlock()

	now := time.Now()
	for key, record := range loginFailures {
		if now.Sub(record.windowAt) >= loginFailWindow {
			delete(loginFailures, key)
		}
	}
	record, ok := loginFailures[ip]
	return ok && record.count >= loginFailLimit
}

func recordLoginFailure(ip string) {
	loginFailMu.Lock()
	defer loginFailMu.Unlock()

	now := time.Now()
	record, ok := loginFailures[ip]
	if !ok || now.Sub(record.windowAt) >= loginFailWindow {
		loginFailures[ip] = &loginFailRecord{count: 1, windowAt: now}
		return
	}
	record.count++
}

func clearLoginFailures(ip string) {
	loginFailMu.Lock()
	delete(loginFailures, ip)
	loginFailMu.Unlock()
}
