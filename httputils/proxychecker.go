package httputils

import (
	"math/rand"
	"net/http"
	"sync"
	"time"
)

var TriggerTimeout = 3 * time.Second

var ProxyTestUrls = []string{
	"https://www.baidu.com/robots.txt",
	"https://www.qq.com/robots.txt",
	"https://www.google.com/favicon.ico",
	"https://bbs.com/templates/standard/css/jquery.simpleLens.css",
	"https://bbs.com/robots.txt",
	"https://twitter.com/favicon.ico",
	"https://youtube.com/favicon.ico",
	"https://www.hao123.com/robots.txt"}

type TrafficController struct {
	workingCount int
	lock         sync.RWMutex
	content      []bool
}

func NewTrafficController(workingCount int) *TrafficController {
	traffic := &TrafficController{workingCount: workingCount, lock: sync.RWMutex{}, content: make([]bool, workingCount)}
	for i := 0; i < traffic.workingCount; i++ {
		traffic.content[i] = false
	}
	return traffic
}

func (traffic *TrafficController) TestAndWork() bool {
	mayWork := false
	traffic.lock.RLock()
	for i := 0; i < traffic.workingCount; i++ {
		if !traffic.content[i] {
			mayWork = true
			break
		}
	}
	traffic.lock.RUnlock()
	
	if !mayWork {
		return false
	}
	
	traffic.lock.Lock()
	defer traffic.lock.Unlock()
	for i := 0; i < traffic.workingCount; i++ {
		if !traffic.content[i] {
			traffic.content[i] = true
			return true
		}
	}
	return false
}

func (traffic *TrafficController) WorkDone() {
	traffic.lock.Lock()
	defer traffic.lock.Unlock()
	for i := 0; i < traffic.workingCount; i++ {
		if traffic.content[i] {
			traffic.content[i] = false
			return
		}
	}
}

func ProxyTriggerWithController(proxyUrl string, url string, controller *TrafficController) {
	if !controller.TestAndWork() {
		return
	}
	defer controller.WorkDone()
	
	ProxyTrigger(proxyUrl, url)
}

func ProxyTrigger(proxyUrl string, url string) (bool, error) {
	if len(url) == 0 {
		url = ProxyTestUrls[rand.Intn(len(ProxyTestUrls))]
	}
	resp, err := BasicRequestGet(url, proxyUrl, TriggerTimeout)
	ForceCloseResponse(resp)
	
	if err != nil {
		return false, err
	}
	return true, nil
}

func CreateProxyCheckRequest(url string) *http.Request {
	if len(url) == 0 {
		url = ProxyTestUrls[rand.Intn(len(ProxyTestUrls))]
	}
	
	req, _ := CreateHttpRequest("GET", url, nil, nil)
	return req
}

func ProxyTriggerWithLocalAddr(proxyUrl string, url string, localAddr string, timeout time.Duration) (bool, error) {
	if len(url) == 0 {
		url = ProxyTestUrls[rand.Intn(len(ProxyTestUrls))]
	}
	if timeout <= 0 {
		timeout = TriggerTimeout
	}
	resp, err := GetResponse("GET", url, nil, proxyUrl, timeout, nil, localAddr)
	ForceCloseResponse(resp)
	
	if err != nil {
		return false, err
	}
	return true, nil
}
