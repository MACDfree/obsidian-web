package middleware

import (
	"obsidian-web/log"
	"os"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/memstore"
	"github.com/gin-gonic/gin"
)

var (
	authKey = os.Getenv("key1")
	encKey  = os.Getenv("key2")
)

func Session(name string) gin.HandlerFunc {
	authKey = "12345678901234561234567890123456"
	encKey = "12345678901234561234567890123456"

	if authKey == "" || encKey == "" {
		log.Fatalf("authKey or encKey is empty")
	}
	log.Infof("key1: %s\tkey2: %s", authKey, encKey)
	store := memstore.NewStore([]byte(authKey), []byte(encKey))
	return sessions.Sessions(name, store)
}
