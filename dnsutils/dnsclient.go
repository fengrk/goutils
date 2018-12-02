package dnsutils

import (
	"github.com/frkhit/goutils/httputils"
	"github.com/frkhit/logger"
	"strings"
	"time"
)

func GetRemoteList(remoteStr string) []string {
	// prepare remoteList
	var remoteList []string
	for _, remote := range strings.Split(remoteStr, ",") {
		if len(remote) < 5 {
			continue
		}
		if len(strings.Split(remote, ":")) == 1 {
			remoteList = append(remoteList, remote+":53")
		} else {
			remoteList = append(remoteList, remote)
		}
	}
	if len(remoteList) == 0 {
		remoteList = append(remoteList, "223.5.5.5:53", "223.6.6.6:53")
	}
	return remoteList
}

func StartDNSSimpleServer(hostPathOrUri, dnsServer, dnsType, addr string, port int) {
	// start host file refresh worker and get local host file
	hostFile := hostPathOrUri
	tmpHost := ""
	if strings.Index(hostPathOrUri, "https://") == 0 || strings.Index(hostPathOrUri, "http://") == 0 {
		StartHostFileRefreshWorker(hostPathOrUri, 15*time.Minute)
		if len(GetDownloadedHostFile()) > 0 {
			hostFile = GetDownloadedHostFile()
		} else {
			tmpHost = GetLogPath("tmp.host.log")
			if downloadErr := httputils.DownloadFile(hostPathOrUri, tmpHost); downloadErr != nil {
				logger.Errorf("fail to download host file from uri: %s\n", hostPathOrUri)
				hostFile = ""
			} else {
				hostFile = tmpHost
			}
		}
	}
	
	// prepare remoteList
	remoteList := GetRemoteList(dnsServer)
	
	// start dns server
	switch dnsType {
	case "dnsmasq":
		StartDNSMASQ(addr, port, hostFile, remoteList)
	default:
		SafeCloseDNSMASQ()
		StartDNSServer(addr, port, hostFile, remoteList)
	}
}
