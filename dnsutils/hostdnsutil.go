package dnsutils

import (
	"github.com/frkhit/goutils/common"
	"github.com/frkhit/goutils/httputils"
	"github.com/frkhit/logger"
	"time"
)

const TargetHostUrl = "https://raw.githubusercontent.com/googlehosts/hosts/master/hosts-files/hosts"

var hostDNSUtil *HostDNSUtil = nil

type HostDNSUtil struct {
	uri                   string
	hostFile              string
	refresh               time.Duration
	hostRecordTriggerList []func(map[string]string)
	hostFileTriggerList   []func(string)
}

func (util *HostDNSUtil) updateHost() {
	if util.refresh > 0 && len(util.uri) > 0 {
		oldMd5, _ := common.GetMd5FromFile(util.hostFile)
		if err := httputils.DownloadFile(util.uri, util.hostFile); err != nil {
			logger.Errorln("fail to download host: ", err)
			return
		}
		newMd5, err := common.GetMd5FromFile(util.hostFile)
		if err != nil {
			logger.Errorf("fail to get md5 of file %s, error is %s\n", util.hostFile, err)
			return
		}
		if newMd5 == oldMd5 {
			logger.Infoln("host not change, no need to update!")
			return
		}
		
		// deal with new host file
		if len(util.hostFileTriggerList) > 0 {
			for _, handler := range util.hostFileTriggerList {
				handler(util.hostFile)
			}
		}
		
		// deal with new host record
		hostIPRecord, hostErr := common.ParseHostFile(util.hostFile)
		if hostErr == nil && len(util.hostRecordTriggerList) > 0 {
			for _, handler := range util.hostRecordTriggerList {
				tmpHostIPRecord := make(map[string]string, len(hostIPRecord))
				for key, value := range hostIPRecord {
					tmpHostIPRecord[key] = value
				}
				handler(tmpHostIPRecord)
			}
		}
	}
}

func (util *HostDNSUtil) AddHostRecordUpdateTrigger(callbackHandler func(map[string]string)) {
	if callbackHandler != nil {
		util.hostRecordTriggerList = append(util.hostRecordTriggerList, callbackHandler)
	}
}

func (util *HostDNSUtil) AddHostFileUpdateTrigger(callbackHandler func(string)) {
	if callbackHandler != nil {
		util.hostFileTriggerList = append(util.hostFileTriggerList, callbackHandler)
	}
}

func (util *HostDNSUtil) loopRefresh() {
	for {
		util.updateHost()
		time.Sleep(util.refresh)
	}
}

func StartHostFileRefreshWorker(uri string, refreshTime time.Duration) {
	if hostDNSUtil == nil {
		hostDNSUtil = &HostDNSUtil{uri: uri, refresh: refreshTime, hostFile: GetLogPath("tmp.hosts.log"), hostRecordTriggerList: []func(map[string]string){}}
	}
	go hostDNSUtil.loopRefresh()
}

func AddHostRecordUpdateTrigger(callbackHandler func(map[string]string)) {
	if hostDNSUtil == nil {
		logger.Fatalf("hostDNSUtil not exist now!")
	}
	hostDNSUtil.AddHostRecordUpdateTrigger(callbackHandler)
}

func AddHostFileUpdateTrigger(callbackHandler func(string)) {
	if hostDNSUtil == nil {
		logger.Fatalf("hostDNSUtil not exist now!")
	}
	hostDNSUtil.AddHostFileUpdateTrigger(callbackHandler)
}

func GetDownloadedHostFile() string {
	if hostDNSUtil == nil {
		return ""
	}
	if common.FileExists(hostDNSUtil.hostFile) {
		return hostDNSUtil.hostFile
	}
	return ""
}
