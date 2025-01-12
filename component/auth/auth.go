package auth

import (
	"strconv"
	"sync"
)

var authInfos = make(map[string]AuthUser)

type Authenticator interface {
	Verify(user string, pass string) bool
	VerifyPort(port int, user string, pass string) bool
	Users() []string
}

type AuthUser struct {
	User string
	Pass string
}

type inMemoryAuthenticator struct {
	storage   *sync.Map
	usernames []string
}

func (au *inMemoryAuthenticator) Verify(user string, pass string) bool {
	realPass, ok := au.storage.Load(user)
	return ok && realPass == pass
}
func AddAuth(port int, user, pass string) {
	item := AuthUser{
		User: user,
		Pass: pass,
	}
	authInfos[strconv.Itoa(port)] = item
}
func (au *inMemoryAuthenticator) VerifyPort(port int, user string, pass string) bool {
	if item, ok := authInfos[strconv.Itoa(port)]; ok {
		return user == item.User && pass == item.Pass
	} else {
		return au.Verify(user, pass)
	}
	return false
}

func (au *inMemoryAuthenticator) Users() []string { return au.usernames }

func NewAuthenticator(users []AuthUser, authes map[string]AuthUser) Authenticator {
	authInfos = authes
	if len(users) == 0 {
		return nil
	}

	au := &inMemoryAuthenticator{storage: &sync.Map{}}
	for _, user := range users {
		au.storage.Store(user.User, user.Pass)
	}
	usernames := make([]string, 0, len(users))
	au.storage.Range(func(key, value interface{}) bool {
		usernames = append(usernames, key.(string))
		return true
	})
	au.usernames = usernames

	return au
}
