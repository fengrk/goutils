package dnsutils

import (
	"fmt"
	"github.com/frkhit/goutils/common"
	"github.com/frkhit/goutils/executils"
	"github.com/frkhit/logger"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const TargetHost = "/etc/usr_hosts"

func backupFile(src, dst string) error {
	if !common.FileExists(src) {
		return nil
	}
	return common.CopyFile(src, dst)
}

func writeStrListToFile(strList []string, targetFile string, mode os.FileMode) error {
	return common.WriteStringToFile(strings.Join(strList, "\n")+"\n\n", targetFile, mode, false)
}

func ExecDNSMASQ(cmd string) error {
	return exec.Command("/bin/sh", "-c", "sudo /etc/init.d/dnsmasq "+cmd).Run()
}

func createResolveConf(localAddr string, remoteList []string) error {
	if err := backupFile("/etc/resolv.dnsmasq.conf", GetLogPath(common.GetUUID()+"resolv.dnsmasq.conf")); err != nil {
		return fmt.Errorf("fail to backup /etc/resolv.dnsmasq.conf")
	}
	if err := backupFile("/etc/resolv.conf", GetLogPath(common.GetUUID()+"resolv.conf")); err != nil {
		return fmt.Errorf("fail to backup /etc/resolv.conf")
	}
	
	strList := []string{"nameserver " + localAddr}
	for _, remote := range remoteList {
		strList = append(strList, "nameserver "+remote)
	}
	
	if err := writeStrListToFile(strList, "/etc/resolv.conf", 0644); err != nil {
		return fmt.Errorf("fail to edit /etc/resolv.conf")
	}
	
	var strList2 []string
	for _, remote := range remoteList {
		strList2 = append(strList2, "nameserver "+remote)
	}
	if err := writeStrListToFile(strList2, "resolv.dnsmasq.conf", 0644); err != nil {
		return fmt.Errorf("fail to edit resolv.dnsmasq.conf")
	}
	return nil
}

func createDNSMASQConf(addr string, port int, hostFile string) error {
	if err := backupFile("/etc/dnsmasq.conf", GetLogPath(common.GetUUID()+"dnsmasq.conf")); err != nil {
		return fmt.Errorf("fail to backup /etc/dnsmasq.conf")
	}
	strList := []string{
		"port=" + strconv.Itoa(port),
		"domain-needed",
		"bogus-priv",
		"resolv-file=/etc/resolv.dnsmasq.conf",
		"strict-order",
		"listen-address=" + addr,
		"expand-hosts",
		"addn-hosts=" + TargetHost,
	}
	if err := writeStrListToFile(strList, "dnsmasq.conf", 0644); err != nil {
		return fmt.Errorf("fail to edit resolv.dnsmasq.conf")
	}
	
	if common.FileExists(hostFile) {
		err := common.CopyFile(hostFile, TargetHost)
		if err != nil {
			logger.Errorf("fail to copy %s to %s, error is %s\n", hostFile, TargetHost, err)
		}
	} else {
		logger.Infof("host file is empty, clear %s\n", TargetHost)
		err := writeStrListToFile([]string{"\n"}, TargetHost, 0644)
		if err != nil {
			logger.Errorf("fail to clear %s, error is %s\n", TargetHost, err)
		}
	}
	
	return nil
}

func StartDNSMASQ(addr string, port int, hostFile string, remoteList []string) {
	if runtime.GOOS == "windows" {
		logger.Fatalln("dnsmasq would not start in windows!")
	}
	
	// prepare conf
	if err := createDNSMASQConf(addr, port, hostFile); err != nil {
		logger.Fatal(err)
	}
	if err := createResolveConf(addr+":"+strconv.Itoa(port), remoteList); err != nil {
		logger.Fatal(err)
	}
	
	// start dnsmasq
	err := ExecDNSMASQ("restart")
	if err != nil {
		logger.Fatalf("fail to start dnsmasq, error is %s", err)
	} else {
		logger.Infoln("success to start dnsmasq!")
	}
	
	// attach new host record trigger
	if hostDNSUtil != nil {
		hostDNSUtil.AddHostRecordUpdateTrigger(func(record map[string]string) {
			var strList []string
			for host, ip := range record {
				strList = append(strList, host+"\t"+ip)
			}
			if err := writeStrListToFile(strList, TargetHost, 0644); err != nil {
				logger.Errorln("fail to update host file: ", err)
				return
			}
			logger.Infoln("success to update host:", TargetHost)
		})
	}
	
	// loop
	for {
		time.Sleep(time.Hour * 1)
	}
}

func SafeCloseDNSMASQ() {
	if runtime.GOOS == "windows" {
		return
	}
	if !executils.ProcessIsRunning("dnsmasq") {
		return
	}
	for {
		ExecDNSMASQ("stop")
		time.Sleep(2 * time.Second)
		if !executils.ProcessIsRunning("dnsmasq") {
			return
		}
	}
}
