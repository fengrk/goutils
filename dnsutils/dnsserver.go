package dnsutils

// ref: https://blog.csdn.net/yatere/article/details/43318147, by Yatere
// ref: http://mkaczanowski.com/golang-build-dynamic-dns-service-go/, by Mateusz Kaczanowski
import (
	"encoding/json"
	"fmt"
	"github.com/frkhit/goutils/common"
	"github.com/frkhit/logger"
	"github.com/miekg/dns"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	DNSPort                              = 53
	DNSDefaultTTL                        = 1 * time.Minute
	DNSFailTTL                           = 15 * time.Second
	LongLiveDNSTTL         time.Duration = 0
	keyListCacheKey                      = "DNSSimpleServerCacheKeyList"
	keyListSep                           = ">>>|<<<"
	DNSQueryDefaultTimeout               = 5 * time.Second
	DNSDefaultRType        uint16        = 0
)

type CacheContent struct {
	TTL   time.Duration `json:"ttl"`
	Value string        `json:"value"`
}

type CacheContentRecord struct {
	Record map[uint16]*CacheContent `json:"record"`
}

type DNSSimpleServer struct {
	failRecord     map[string]time.Duration
	failRecordLock sync.RWMutex
	remoteList     []string
	remote         string
	dbCache        DBCache
	ttl            time.Duration
}

func (ds *DNSSimpleServer) Close() {
	if ds.dbCache != nil {
		ds.dbCache.Close()
	}
}

func (ds *DNSSimpleServer) UpdateHostRecord(record map[string]string) {
	updateFunc := func(newRecord map[string]string) {
		logger.Infof("trying to update host record, %d record would be use\n", len(newRecord))
		
		var oldCacheKeyList, newCacheKeyList []string
		cacheRecord := make(map[string]string)
		
		// find record from hostIPRecord
		for domain, ip := range newRecord {
			domain = strings.TrimRight(domain, ".") + "."
			if len(domain) < 3 {
				continue
			}
			cacheKey := ds.getKey(domain)
			rr, err := dns.NewRR(fmt.Sprintf("%s A %s", domain, ip))
			if err != nil {
				logger.Errorf("fail to create record: domain[%s], ip[%s], error is %s", domain, ip, err)
				continue
			}
			cacheRecord[cacheKey] = rr.String()
		}
		
		// del old host record
		if oldKeyListStr, err := ds.dbCache.Get(keyListCacheKey); err == nil {
			for _, key := range strings.Split(oldKeyListStr, keyListSep) {
				if len(key) > 2 {
					oldCacheKeyList = append(oldCacheKeyList, key)
				}
			}
		}
		if len(oldCacheKeyList) > 0 {
			logger.Infof("there are %d old key list in dbCache, trying to delete them...", len(oldCacheKeyList))
			delErr := ds.dbCache.BatchDelete(oldCacheKeyList)
			if delErr != nil {
				logger.Fatalf("fail to run `BatchDelete`, error is %s", delErr)
			}
			logger.Infoln("success to delete old key in dbCache!")
		}
		
		// save new host record
		if len(cacheRecord) > 0 {
			logger.Infof("trying to save %d new host ip record in dbCache...", len(cacheRecord))
			for key := range cacheRecord {
				newCacheKeyList = append(newCacheKeyList, key)
			}
			
			setErr := ds.setBatchValue(cacheRecord, LongLiveDNSTTL, true)
			if setErr != nil {
				logger.Fatalf("fail to run `BatchSet`, error is %s", setErr)
			}
			
			setErr = ds.dbCache.Set(keyListCacheKey, strings.Join(newCacheKeyList, keyListSep))
			if setErr != nil {
				logger.Fatalf("fail to save keyListCacheKey, error is %s", setErr)
			}
			logger.Infoln("success to save all new host ip in dbCache!")
		}
		logger.Infof("success to update domain record, current record is %d\n", len(cacheRecord))
	}
	
	// run in daemon
	go updateFunc(record)
}

func (ds *DNSSimpleServer) getKey(domain string) (string) {
	if n, ok := dns.IsDomainName(domain); ok {
		labels := dns.SplitDomainName(domain)
		
		var tmp string
		for i := 0; i < int(math.Floor(float64(n/2))); i++ {
			tmp = labels[i]
			labels[i] = labels[n-1-i]
			labels[n-1-i] = tmp
		}
		
		return strings.Join(labels, ".")
	} else {
		logger.Errorln("Invalid domain: " + domain)
	}
	return domain
}

func (ds *DNSSimpleServer) getRecord(domain string, rType uint16) (rList []dns.RR, err error) {
	// find record from bucket
	cacheKey := ds.getKey(domain)
	realType := rType
	result, err := ds.getResult(cacheKey)
	if err != nil {
		return rList, fmt.Errorf("key[%s] not found in record", cacheKey)
	}
	cacheContent, exists := result[rType]
	if !exists {
		cacheContent, exists = result[DNSDefaultRType]
		realType = DNSDefaultRType
	}
	if !exists {
		err = fmt.Errorf("key[%s] found in record, but rType[%d,%d] not found in result", cacheKey, rType, DNSDefaultRType)
	}
	if err == nil {
		var tmpList []dns.RR
		for _, rrStr := range strings.Split(cacheContent.Value, keyListSep) {
			if len(rrStr) > 0 {
				r, rErr := dns.NewRR(rrStr)
				if rErr != nil {
					delete(result, realType)
					ds.setResult(cacheKey, result)
					return rList, fmt.Errorf("fail to create dns.RR from string, error is %s", rErr)
				} else {
					tmpList = append(tmpList, r)
				}
			}
		}
		if len(tmpList) > 0 {
			return tmpList, nil
		} else {
			delete(result, realType)
			ds.setResult(cacheKey, result)
			err = fmt.Errorf("fail to get valid dns.RR from string")
		}
	}
	
	return rList, err
}

func (ds *DNSSimpleServer) getResult(key string) (map[uint16]*CacheContent, error) {
	value, err := ds.dbCache.Get(key)
	if err != nil {
		return nil, err
	}
	
	data := CacheContentRecord{}
	err = json.Unmarshal([]byte(value), &data)
	if err != nil {
		ds.dbCache.Delete(key)
		return nil, err
	}
	
	var clearRType []uint16
	currentTime := time.Duration(time.Now().Unix()) * time.Second
	for rType, cacheContent := range data.Record {
		if cacheContent.TTL > LongLiveDNSTTL && cacheContent.TTL < currentTime { // timeout
			clearRType = append(clearRType, rType)
		}
	}
	if len(clearRType) > 0 {
		for _, rType := range clearRType {
			delete(data.Record, rType)
		}
		ds.setResult(key, data.Record)
	}
	
	return data.Record, nil
}

func (ds *DNSSimpleServer) setResult(key string, result map[uint16]*CacheContent) (error) {
	data := &CacheContentRecord{Record: result}
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("fail to dump to json: %s", err)
	}
	return ds.dbCache.Set(key, string(jsonBytes))
}

func (ds *DNSSimpleServer) setBatchValue(record map[string]string, ttl time.Duration, isHostRecord bool) (error) {
	var err error
	currentTTL := LongLiveDNSTTL
	if ttl > 0 {
		currentTTL = time.Duration(time.Now().Unix())*time.Second + ttl
	}
	for key, value := range record {
		result, err := ds.getResult(key)
		if err != nil && result == nil {
			result = make(map[uint16]*CacheContent)
		}
		result[dns.TypeA] = &CacheContent{TTL: currentTTL, Value: value}
		if isHostRecord {
			result[DNSDefaultRType] = &CacheContent{TTL: currentTTL, Value: value}
		}
		err = ds.setResult(key, result)
		if err != nil {
			return fmt.Errorf("fail to store record: %s", err)
		}
	}
	return err
}

func (ds *DNSSimpleServer) updateRecord(rList []dns.RR, q *dns.Question) {
	defer func() {
		if e := recover(); e != nil {
			logger.Errorf("updateRecord panic %s\n", e)
		}
	}()
	name := q.Name
	cacheKey := ds.getKey(name)
	result, err := ds.getResult(cacheKey)
	if err != nil && result == nil {
		result = make(map[uint16]*CacheContent)
	}
	
	if len(rList) == 0 {
		// del record
		delete(result, q.Qtype)
	} else {
		// add record
		var answerStrList []string
		for _, r := range rList {
			answerStrList = append(answerStrList, r.String())
		}
		result[q.Qtype] = &CacheContent{TTL: DNSDefaultTTL + time.Duration(time.Now().Unix())*time.Second, Value: strings.Join(answerStrList, keyListSep)}
	}
	
	err = ds.setResult(cacheKey, result)
	if err != nil {
		logger.Errorf("fail to store cacheKey[%s]: %s\n", cacheKey, err)
	}
}

func (ds *DNSSimpleServer) realQuery(r *dns.Msg, m *dns.Msg, fn func(r, m, newMsg *dns.Msg)) {
	var newMsg *dns.Msg
	var err error
	for _, remote := range ds.remoteList {
		c := new(dns.Client)
		c.Timeout = DNSQueryDefaultTimeout
		c.Net = "udp"
		newMsg, _, err = c.Exchange(r, remote)
		if err != nil {
			if newMsg != nil && newMsg.Question != nil && len(newMsg.Question) > 0 {
				logger.Errorf("fail to query ip for domain[%s] from remote server[%s]: error is %s\n", newMsg.Question[0].Name, remote, err)
			} else {
				logger.Errorf("fail to query ip for domain from remote server[%s]: error is %s\n", remote, err)
			}
			newMsg = nil
			continue
		}
		
		if len(newMsg.Answer) == 0 {
			if len(newMsg.Question) > 0 {
				logger.Errorf("fail to query ip for domain[%s] from remote server[%s]\n", newMsg.Question[0].Name, remote)
			} else {
				logger.Errorf("fail to query ip for domain from remote server[%s]\n", remote)
			}
			newMsg = nil
			continue
		}
		break
	}
	fn(r, m, newMsg)
}

func (ds *DNSSimpleServer) parseQuery(r *dns.Msg, m *dns.Msg) {
	switch len(m.Question) {
	case 0:
		logger.Errorln("Query Error: question cannot be null!")
	case 1:
		question := m.Question[0]
		answerList, e := ds.getRecord(question.Name, question.Qtype)
		if e == nil {
			m.Answer = append(m.Answer, answerList...)
			logger.Infof("host[%s] found in record\n", question.Name)
			return
		}
		logger.Errorf("host[%s] not found in record: error is %s\n", question.Name, e)
		cacheKey := ds.getKey(question.Name)
		ds.failRecordLock.RLock()
		ttl, exists := ds.failRecord[cacheKey]
		ds.failRecordLock.RUnlock()
		
		if !exists || ttl < time.Duration(time.Now().Unix())*time.Second {
			ds.realQuery(r, m, func(r, m, newMsg *dns.Msg) {
				isFail := false
				if newMsg != nil {
					m.Answer = append(m.Answer, newMsg.Answer...)
					ds.updateRecord(newMsg.Answer, &newMsg.Question[0])
				} else {
					// not found ip from remote dns server
					//q := m.Question[0]
					//localhostRR, _ := dns.NewRR(fmt.Sprintf("%s A 127.0.0.1", q.Name))
					//ds.updateRecord([]dns.RR{localhostRR}, &q)
					isFail = true
				}
				if isFail {
					ds.failRecordLock.Lock()
					ds.failRecord[cacheKey] = time.Duration(time.Now().Unix())*time.Second + DNSFailTTL
					ds.failRecordLock.Unlock()
				} else {
					if exists {
						ds.failRecordLock.Lock()
						delete(ds.failRecord, cacheKey)
						ds.failRecordLock.Unlock()
					}
				}
			})
		}
	
	default:
		// multi question: not support in practice
		ds.realQuery(r, m, func(r, m, newMsg *dns.Msg) {
			if newMsg != nil {
				m.Answer = append(m.Answer, newMsg.Answer...)
			}
		})
	}
}

func (ds *DNSSimpleServer) handleDnsRequest(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = false
	
	switch r.Opcode {
	case dns.OpcodeQuery:
		ds.parseQuery(r, m)
	
	case dns.OpcodeUpdate:
		for _, question := range r.Question {
			for _, rr := range r.Ns {
				ds.updateRecord([]dns.RR{rr}, &question)
			}
		}
	}
	
	if r.IsTsig() != nil {
		if w.TsigStatus() == nil {
			m.SetTsig(r.Extra[len(r.Extra)-1].(*dns.TSIG).Hdr.Name,
				dns.HmacMD5, 300, time.Now().Unix())
		} else {
			logger.Infoln("Status ", w.TsigStatus().Error())
		}
	}
	w.WriteMsg(m)
}

func (ds *DNSSimpleServer) StartDNSServer(addr string, port int) {
	// attach new host record trigger
	if hostDNSUtil != nil {
		hostDNSUtil.AddHostRecordUpdateTrigger(ds.UpdateHostRecord)
	}
	
	// attach request handler func
	dns.HandleFunc(".", ds.handleDnsRequest)
	
	// start server
	server := &dns.Server{Addr: addr + ":" + strconv.Itoa(port), Net: "udp"}
	logger.Infof("Starting at %s:%d\n", addr, port)
	
	err := server.ListenAndServe()
	defer ds.Close()
	defer server.Shutdown()
	
	if err != nil {
		logger.Fatalf("Failed to setup the udp server: %s\n", err.Error())
	}
}

func StartDNSServer(addr string, port int, hostFile string, remoteList []string) {
	// bdb
	dbCache := NewMemCache(GetLogPath("data.db"))
	
	// hostIpRecord
	hostIpRecord := map[string]string{}
	if hostFile != "" {
		var recordErr error
		hostIpRecord, recordErr = common.ParseHostFile(hostFile)
		if recordErr != nil {
			logger.Errorln("fail to parse host file[%s], error is %s\n", hostFile, recordErr)
		}
	}
	
	if port <= 0 {
		port = DNSPort
	}
	var newRemoteList []string
	for _, host := range remoteList {
		if len(host) > 5 {
			newRemoteList = append(newRemoteList, host)
		}
	}
	ds := &DNSSimpleServer{remoteList: remoteList, remote: remoteList[0], dbCache: dbCache, ttl: DNSDefaultTTL, failRecord: make(map[string]time.Duration), failRecordLock: sync.RWMutex{}}
	if len(hostIpRecord) > 0 {
		ds.UpdateHostRecord(hostIpRecord)
	}
	
	ds.StartDNSServer(addr, port)
}
