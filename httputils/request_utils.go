package httputils

import (
	"bytes"
	"fmt"
	"github.com/frkhit/logger"
	"golang.org/x/net/html/charset"
	"golang.org/x/net/proxy"
	"golang.org/x/text/transform"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	DefaultLocalAddr   = "127.0.0.1"
	DefaultLocalPort   = 1080
	ProxyUrlFormat     = "socks5://%s:%d"
	MaxIdleConnections = 64
)

var (
	DefaultProxyUrl          = fmt.Sprintf(ProxyUrlFormat, DefaultLocalAddr, DefaultLocalPort)
	DefaultTimeout           = 15 * time.Second
	DefaultGlobalClientCache = NewGlobalClientCache()
)

type GlobalClientCache struct {
	globalResolver *net.Resolver
	clientCache    map[string]*http.Client
	transportCache map[string]*http.Transport
	lock           sync.RWMutex
}

func NewGlobalClientCache() *GlobalClientCache {
	return &GlobalClientCache{
		globalResolver: nil,
		clientCache:    make(map[string]*http.Client),
		transportCache: make(map[string]*http.Transport),
		lock:           sync.RWMutex{},
	}
}

func (client *GlobalClientCache) CreateClientId(proxyAddr string, timeout time.Duration, bindLocalAddr string) string {
	return GetMd5(fmt.Sprintf("%s:::%s:::%s", proxyAddr, timeout, bindLocalAddr))
}

func (client *GlobalClientCache) CreateTransportId(proxyAddr string, bindLocalAddr string) string {
	return GetMd5(fmt.Sprintf("%s:::%s", proxyAddr, bindLocalAddr))
}

// bindLocalAddr: "", "127.0.0.1", "127.0.0.1:6666"
func (client *GlobalClientCache) CreateHttpClient(proxyAddr string, timeout time.Duration, bindLocalAddr string) (*http.Client, error) {
	clientId := client.CreateClientId(proxyAddr, timeout, bindLocalAddr)
	
	// get client
	client.lock.RLock()
	httpClient, exists := client.clientCache[clientId]
	client.lock.RUnlock()
	
	if exists {
		return httpClient, nil
	}
	
	// create client
	httpClient = &http.Client{}
	
	// get transport
	transportId := client.CreateTransportId(proxyAddr, bindLocalAddr)
	client.lock.RLock()
	transport, exists := client.transportCache[transportId]
	client.lock.RUnlock()
	
	if !exists {
		transport = &http.Transport{DisableKeepAlives: false, MaxIdleConnsPerHost: MaxIdleConnections,}
		
		baseDialer := &net.Dialer{
			//LocalAddr: tcpAddr,
			//Timeout:   30 * time.Second,
			//KeepAlive: 30 * time.Second,
		}
		
		// bind local addr
		if len(bindLocalAddr) > 0 {
			ipAndPort := strings.Split(bindLocalAddr, ":")
			if len(ipAndPort) > 2 {
				return nil, fmt.Errorf("bindLocalAddr [%s] is invalid", bindLocalAddr)
			}
			
			localAddr, err := net.ResolveIPAddr("ip", ipAndPort[0])
			if err != nil {
				return nil, err
			}
			tcpAddr := &net.TCPAddr{IP: localAddr.IP}
			if len(ipAndPort) == 2 {
				port, err := strconv.Atoi(ipAndPort[1])
				if err != nil {
					return nil, fmt.Errorf("bindLocalAddr [%s] is invalid", bindLocalAddr)
				}
				tcpAddr.Port = port
			}
			baseDialer.LocalAddr = tcpAddr
		}
		
		// proxy
		var proxyDialer proxy.Dialer
		if len(proxyAddr) > 0 {
			proxyUrl, err := url.Parse(proxyAddr)
			if err != nil {
				return nil, fmt.Errorf("error found in proxyAddr[%s]: %v", proxyAddr, err)
			}
			
			// socks5, http, https: use baseDialer and proxyDialer
			proxyDialer, err = proxy.FromURL(proxyUrl, baseDialer)
			if err != nil {
				return nil, fmt.Errorf("error found in proxyAddr[%s]: %v", proxyAddr, err)
			}
		}
		if proxyDialer != nil {
			transport.Dial = proxyDialer.Dial
		} else {
			transport.Dial = baseDialer.Dial
		}
		
		// update transport
		client.lock.Lock()
		client.transportCache[transportId] = transport
		client.lock.Unlock()
	}
	
	// set transport
	httpClient.Transport = transport
	
	// set timeout
	if timeout > 0 {
		httpClient.Timeout = timeout
	}
	
	// update cache
	client.lock.Lock()
	client.clientCache[clientId] = httpClient
	client.lock.Unlock()
	
	return httpClient, nil
}

func (client *GlobalClientCache) SetGlobalDNSResolver() {
	if client.globalResolver != nil {
		net.DefaultResolver = client.globalResolver
	}
	
	// create resolver
	// todo not finish now
	client.globalResolver = net.DefaultResolver
}

func GetIPAndPort(url *url.URL) (string, int) {
	host := strings.Split(url.Host, ":")[0]
	
	portStr := url.Port()
	port := 80
	if len(portStr) == 0 {
		if url.Scheme == "https" {
			port = 443
		}
	} else {
		port, _ = strconv.Atoi(portStr)
	}
	
	ips, _ := net.LookupIP(host)
	if len(ips) == 0 {
		// todo deal with error
		return "", port
	}
	
	return string(ips[0]), port
}

func SetGlobalDnsCache() {
	// set default client for http package
	DefaultGlobalClientCache.SetGlobalDNSResolver()
}

func CreateHttpRequest(method string, uri string, headers map[string]string, data io.Reader) (*http.Request, error) {
	// req data
	req, err := http.NewRequest(method, uri, data)
	if err != nil {
		return nil, nil
	}
	
	// headers
	if headers == nil {
		headers = make(map[string]string)
	}
	_, isPresent := headers["User-Agent"]
	if !isPresent {
		headers["User-Agent"] = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/69.0.3497.100 Safari/537.36"
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	return req, nil
}

func GetResponse(method string, uri string, headers map[string]string, proxyAddr string, timeout time.Duration, data io.Reader, bindAddr string) (*http.Response, error) {
	logger.Infof("<REQ-DEBUG> getting response for method=[%s], uri=[%s], headers=[%s], proxyAddr=[%s], timeout=[%d], data=[%s], bindAddr=[%s]\n", method, uri, headers, proxyAddr, timeout, data, bindAddr)
	client, err := DefaultGlobalClientCache.CreateHttpClient(proxyAddr, timeout, bindAddr)
	if err != nil {
		return nil, err
	}
	
	req, err := CreateHttpRequest(method, uri, headers, data)
	if err != nil {
		return nil, err
	}
	
	// request
	return client.Do(req)
}

func ParseResponse(resp *http.Response) (content string, err error) {
	if resp == nil {
		return "", fmt.Errorf("response cannot be null")
	}
	
	bodyByte, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	
	if err != nil {
		return string(bodyByte), nil
	}
	
	// recreate resp.Body
	resp.Body = ioutil.NopCloser(bytes.NewBuffer(bodyByte))
	defer resp.Body.Close()
	
	// decode body
	e, _, _ := charset.DetermineEncoding(bodyByte, "") // bodyByte changed
	reader := transform.NewReader(resp.Body, e.NewDecoder())
	body, err := ioutil.ReadAll(reader)
	
	return string(body), err
}

func DownloadFile(uri string, imagePath string) (err error) {
	response, err := StrongRequestGet(uri, DefaultProxyUrl, false)
	if response != nil {
		defer response.Body.Close()
	}
	if err != nil {
		return err
	}
	
	//open a file for writing
	file, err := os.Create(imagePath)
	if file != nil {
		defer file.Close()
	}
	if err != nil {
		return err
	}
	
	// Use io.Copy to just dump the response body to the file. This supports huge files
	_, err = io.Copy(file, response.Body)
	return err
}

func BasicRequestGet(url string, proxy string, timeout time.Duration, ) (*http.Response, error) {
	return GetResponse("GET", url, nil, proxy, timeout, nil, "")
}

func SimpleRequestGet(url string) (*http.Response, error) {
	return BasicRequestGet(url, "", DefaultTimeout)
}

func ProxyRequestGet(url string, proxy string) (*http.Response, error) {
	return BasicRequestGet(url, proxy, DefaultTimeout)
}

func RequestGetByWebApi(url string) (*http.Response, error) {
	headers := map[string]string{
		"DNT":          "1",
		"Accept":       "text/plain, */*; q=0.01",
		"Content-Type": "application/x-www-form-urlencoded; charset=UTF-8",
		"Origin":       "https://codebeautify.org",
		"Referer":      "https://codebeautify.org/source-code-viewer",
	}
	
	// todo have error
	return GetResponse("POST", "https://codebeautify.com/URLService", headers, "", DefaultTimeout, strings.NewReader("path="+url), "")
}

func StrongRequestGet(url string, proxy string, useWebAPI bool) (*http.Response, error) {
	var resp1, resp2 *http.Response
	var err1, err2 error
	
	// step 1
	resp1, err1 = SimpleRequestGet(url)
	// success or this is end step
	if err1 == nil || (len(proxy) == 0 && !useWebAPI) {
		return resp1, err1
	}
	
	if resp1 != nil {
		defer resp1.Body.Close()
	}
	logger.Errorln(err1)
	
	// step 2
	if len(proxy) > 0 {
		resp2, err2 = ProxyRequestGet(url, proxy)
		// success or this is end step
		if err2 == nil || !useWebAPI {
			return resp2, err2
		}
		if resp2 != nil {
			defer resp2.Body.Close()
		}
		logger.Errorln(err2)
	}
	
	// step 3
	return RequestGetByWebApi(url)
}

func GetRedirectLocation(uri string) (string) {
	response, _ := SimpleRequestGet(uri)
	if response != nil {
		defer response.Body.Close()
	}
	
	if response == nil {
		return uri
	}
	
	// response.location
	urlObj, errUrl := response.Location()
	if errUrl == nil {
		newUrl := urlObj.String()
		if len(newUrl) > 0 {
			return newUrl
		}
	}
	
	// response.Request
	if response.Request != nil && response.Request.URL != nil {
		newUrl := response.Request.URL.String()
		if len(newUrl) > 0 {
			return newUrl
		}
	}
	return uri
}

func ListRedirectLocation(urlList []string) []string {
	if len(urlList) == 0 {
		return urlList
	}
	
	parseLocation := func(uri string, countDown *sync.WaitGroup, resultChan chan string) {
		defer countDown.Done()
		if len(uri) > 0 {
			newUrl := GetRedirectLocation(uri)
			resultChan <- newUrl
		}
	}
	
	// use goroutine
	resultChan := make(chan string, len(urlList)+5)
	var newUrlList []string
	countDown := sync.WaitGroup{}
	countDown.Add(len(urlList))
	
	for _, uri := range urlList {
		go parseLocation(uri, &countDown, resultChan)
	}
	
	countDown.Wait()
	close(resultChan)
	
	// read resultChan
	for {
		newUrl, ok := <-resultChan
		if !ok {
			break
		}
		if len(newUrl) > 0 {
			newUrlList = append(newUrlList, newUrl)
		}
	}
	return newUrlList
}

func ListCanConnectUrls(urlList []string) []string {
	if len(urlList) == 0 {
		return urlList
	}
	
	checkConnection := func(uri string, countDown *sync.WaitGroup, resultChan chan string) {
		defer countDown.Done()
		if len(uri) > 0 {
			resp, err := StrongRequestGet(uri, DefaultProxyUrl, true)
			if err == nil && resp != nil {
				resultChan <- uri
			}
			if resp != nil {
				resp.Body.Close()
			}
		}
	}
	
	// use goroutine
	resultChan := make(chan string, len(urlList)+5)
	var newUrlList []string
	countDown := sync.WaitGroup{}
	countDown.Add(len(urlList))
	
	for _, uri := range urlList {
		go checkConnection(uri, &countDown, resultChan)
	}
	
	countDown.Wait()
	close(resultChan)
	
	// read resultChan
	for {
		newUrl, ok := <-resultChan
		if !ok {
			break
		}
		if len(newUrl) > 0 {
			newUrlList = append(newUrlList, newUrl)
		}
	}
	return newUrlList
}

// ref: https://stackoverflow.com/questions/17948827/reusing-http-connections-in-golang
func ForceCloseResponse(resp *http.Response) {
	if resp != nil {
		io.Copy(ioutil.Discard, resp.Body) // must use this line
		resp.Body.Close()
	}
}

func GetNewUrl(alternateUrls []string, defaultUrl string) string {
	for _, uri := range alternateUrls {
		newUrl := GetRedirectLocation(uri)
		if newUrl != uri {
			return newUrl
		}
	}
	return defaultUrl
}
