package middleware

import (
	"crypto/rand"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/memstore"
	"github.com/gin-gonic/gin"
)

func generateKey() []byte {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		panic(err)
	}
	return key
}

func Session(name string) gin.HandlerFunc {
	store := memstore.NewStore(generateKey(), generateKey())
	return sessions.Sessions(name, store)
}
