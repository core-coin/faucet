package server

import (
	"fmt"
	"git.energy/corepass/corepass-http-helpers/logger"
	"github.com/core-coin/faucet/internal/storage"
	"github.com/golang-jwt/jwt/v4"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/jellydator/ttlcache/v2"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/negroni"

	"github.com/core-coin/faucet/internal/chain"
)

type AuthLimiter struct {
	*logger.Logger

	mutex      sync.Mutex
	cache      *ttlcache.Cache
	proxyCount int
	ttl        time.Duration

	storage storage.Storage
}

func NewAuthLimiter(proxyCount int, ttl time.Duration, storage storage.Storage, logger *logger.Logger) *AuthLimiter {
	cache := ttlcache.NewCache()
	cache.SkipTTLExtensionOnHit(true)
	return &AuthLimiter{
		Logger:     logger,
		storage:    storage,
		cache:      cache,
		proxyCount: proxyCount,
		ttl:        ttl,
	}
}

func (l *AuthLimiter) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	jwtToken := r.PostFormValue(JWTKey)
	token, _ := jwt.Parse(jwtToken, nil)
	sub := token.Claims.(jwt.MapClaims)["sub"]
	if sub.(string) == "" {
		l.DebugW(w, http.StatusBadRequest, "Bad JWT Token", "Bad JWT Token")
		return
	}
	if strings.Split(sub.(string), ":")[0] != "coreid" {
		l.DebugW(w, http.StatusBadRequest, "Bad JWT Token", "Bad JWT Token")
		return
	}
	address := strings.Split(sub.(string), ":")[1]

	if !chain.IsValidAddress(address, true) {
		http.Error(w, "invalid address", http.StatusBadRequest)
		return
	}
	if l.ttl <= 0 {
		next.ServeHTTP(w, r)
		return
	}
	clintIP := getClientIPFromRequest(l.proxyCount, r)

	_, err := l.storage.GetCoreID(address)
	if err != nil {
		if err == storage.ErrorNotFound {
			next.ServeHTTP(w, r)
			if w.(negroni.ResponseWriter).Status() != http.StatusOK {
				l.cache.Remove(address)
				l.cache.Remove(clintIP)
				return
			}
			log.WithFields(log.Fields{
				"address":  address,
				"clientIP": clintIP,
			}).Info("Maximum request limit has been reached")
			return
		}
	}

	l.mutex.Lock()
	if l.limitByKey(w, address) || l.limitByKey(w, clintIP) {
		l.mutex.Unlock()
		return
	}
	l.cache.SetWithTTL(address, true, l.ttl)
	l.cache.SetWithTTL(clintIP, true, l.ttl)
	l.mutex.Unlock()

	next.ServeHTTP(w, r)
	if w.(negroni.ResponseWriter).Status() != http.StatusOK {
		l.cache.Remove(address)
		l.cache.Remove(clintIP)
		return
	}
	log.WithFields(log.Fields{
		"address":  address,
		"clientIP": clintIP,
	}).Info("Maximum request limit has been reached")
}

func (l *AuthLimiter) limitByKey(w http.ResponseWriter, key string) bool {
	if _, ttl, err := l.cache.GetWithTTL(key); err == nil {
		errMsg := fmt.Sprintf("You have exceeded the rate limit. Please wait %s before you try again", ttl.Round(time.Second))
		http.Error(w, errMsg, http.StatusTooManyRequests)
		return true
	}
	return false
}
