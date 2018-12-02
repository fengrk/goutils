# dnsutils: 使用golang搭建简单的DNS服务器

# 1.dnsutils使用示例
## 1.1 启动程序示例
```
package main

import (
	"flag"
	"github.com/frkhit/goutils/dnsutils"
	"github.com/frkhit/goutils/executils"
	"github.com/frkhit/logger"
)

type dnsConfig struct {
	dnsType           string
	dnsServer         string
	hostPathOrUri     string
	useDefaultHostUrl bool
	addr              string
	port              int
}

func getDNSConfig() *dnsConfig {
	config := &dnsConfig{}
	flag.StringVar(&config.dnsType, "type", "golang", "dns server type: golang or dnsmasq")
	flag.StringVar(&config.hostPathOrUri, "host", "", "hosts file path or host uri")
	flag.StringVar(&config.dnsServer, "dns", "", "dns server, like 8.8.8.8,223.5.5.5")
	flag.StringVar(&config.addr, "addr", "127.0.0.1", "dns server binding address")
	flag.IntVar(&config.port, "port", 53, "dns server listening port")
	flag.BoolVar(&config.useDefaultHostUrl, "d", false, "use default host url")
	flag.Parse()
	
	if config.dnsType != "golang" && config.dnsType != "dnsmasq" {
		logger.Fatalf("unknown dnsType[%s]: golang or dnsmasq", config.dnsType)
	}
	
	if config.useDefaultHostUrl {
		config.hostPathOrUri = dnsutils.TargetHostUrl
	}
	
	return config
}

func useDNSSimpleServer() {
	// input hostFile
	config := getDNSConfig()
	
	// start dns server
	dnsutils.StartDNSSimpleServer(config.hostPathOrUri, config.dnsServer, config.dnsType, config.addr, config.port)
}

func main() {
	defer func() {
		if e := recover(); e != nil {
			logger.Errorf("Panic %s\n", e)
		}
	}()
	executils.ShutdownGracefully(func() {
	})
	useDNSSimpleServer()
}
```

## 1.2 WSL中启动DNS服务器
- 启动golang版本的DNS服务器

```
cd ~ && mkdir -p ./log && nohup sudo ./dns.exe -d=false -dns=8.8.8.8 -type=golang >> ./log/all.log 2>&1 &
```
- 或者启动DNSMASQ服务器

```
cd ~ && mkdir -p ./log && nohup sudo ./dns.exe -d=false -dns=8.8.8.8 -type=dnsmasq >> ./log/all.log 2>&1 &
```

# 2.WSL-ubuntu18.04使用dnsmasq
## 2.1.WSL中安装/使用dnsmasq
```
# install
sudo apt-get install dnsmasq
# setting
sudo cp ./cmd/conf/*.conf /etc/

# start
sudo /etc/init.d/dnsmasq start
```

## 2.2.Win10中使用dnsmasq
设置dns主服务器为`127.0.0.1`, 副服务器为`223.5.5.5`

可参考`./cmd/dns.bat`自动设置dns服务器.

## 2.3.Win10开机启动ubuntu中dnsmasq
按照[wsl-autostart](https://github.com/frkhit/wsl-autostart)设置自动启动
