netsh interface ip set dns name="WLAN" source=static addr=127.0.0.1
netsh interface ip set dns name="以太网" source=static addr=127.0.0.1
netsh interface ip add dns name="WLAN" addr=223.5.5.5 index=2  
netsh interface ip add dns name="以太网" addr=223.5.5.5 index=2  



