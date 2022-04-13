package route

import (
	"encoding/json"
	"github.com/Dreamacro/clash/adapter"
	"github.com/Dreamacro/clash/adapter/provider"
	"github.com/Dreamacro/clash/component/auth"
	C "github.com/Dreamacro/clash/constant"
	R "github.com/Dreamacro/clash/rule"
	"github.com/Dreamacro/clash/tunnel"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

var (
	portLock sync.Mutex
)

type startProxyRequest struct {
	Proxy     interface{} `json:"proxy"`
	LocalPort int         `json:"localPort"`
	User      string      `json:"user"`
	Pass      string      `json:"pass"`
}

func proxyManagerRouter() http.Handler {
	r := chi.NewRouter()
	r.Post("/start", startProxy)
	return r
}

func startProxy(writer http.ResponseWriter, request *http.Request) {
	req := startProxyRequest{}
	body, _ := ioutil.ReadAll(request.Body)
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		render.JSON(writer, request, render.M{
			"code": -1,
			"msg":  err.Error(),
		})
		return
	}
	if req.Proxy == nil {
		render.JSON(writer, request, render.M{
			"code": -1,
			"msg":  "proxy is not proxy",
		})
		return
	}
	user := req.User
	pass := req.Pass
	if user == "" {
		user = randString(15)
	}
	if pass == "" {
		pass = randString(15)
	}
	port, err := getFreePort(req.LocalPort)
	if port == 0 || err != nil {
		render.JSON(writer, request, render.M{
			"code": -1,
			"msg":  "无可用端口",
		})
		return
	}
	ok, groupName := createProxy(req.Proxy.(map[string]interface{}))
	if !ok {
		render.JSON(writer, request, render.M{
			"code": -1,
			"msg":  "代理创建失败,请检查proxy参数",
		})
		return
	}
	if ok := createRules("LISTENER-PORT", strconv.Itoa(port), groupName, nil); !ok {
		render.JSON(writer, request, render.M{
			"code": -1,
			"msg":  "开启端口监听失败",
		})
		return
	}
	auth.AddAuth(port, user, pass)
	data := make(map[string]interface{})
	data["scheme"] = "http"
	data["port"] = port
	data["username"] = user
	data["password"] = pass
	render.JSON(writer, request, render.M{
		"code": 0,
		"data": data,
	})
	return
}

func createProxy(newProxy map[string]interface{}) (bool, string) {
	proxy, err := adapter.ParseProxy(newProxy)
	if err != nil {
		return false, ""
	}
	groupname := proxy.Name()
	ps := []C.Proxy{}
	ps = append(ps, proxy)
	hc := provider.NewHealthCheck(ps, "", 0, true)
	pd, _ := provider.NewCompatibleProvider(groupname, ps, hc)
	providers := tunnel.Providers()
	providers[groupname] = pd

	proxies := tunnel.Proxies()
	proxies[groupname] = proxy
	tunnel.UpdateProxies(proxies, providers)
	return true, groupname
}

func createRules(ruleType, payload, target string, params []string) bool {
	parsed, parseErr := R.ParseRule(ruleType, payload, target, params)
	if parseErr != nil {
		return false
	}
	rawRules := tunnel.Rules()
	rawRules = append(rawRules, parsed)
	tunnel.UpdateRules(rawRules)
	return true
}

func randString(len int) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	bytes := make([]byte, len)
	for i := 0; i < len; i++ {
		b := r.Intn(26) + 65
		bytes[i] = byte(b)
	}
	return string(bytes)
}

func getFreePort(port int) (int, error) {
	portLock.Lock()
	defer portLock.Unlock()

	isuse, err := checkPortIsAction(port)
	if isuse == false {
		return port, nil
	}
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
func checkPortIsAction(port int) (bool, error) {
	var err error

	tcpAddress, err := net.ResolveTCPAddr("tcp4", ":"+strconv.Itoa(port))
	if err != nil {
		return true, err
	}
	_, err = net.ListenTCP("tcp", tcpAddress)
	if err != nil {
		return true, err
	}
	return false, nil
}
