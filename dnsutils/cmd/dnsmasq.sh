#!/bin/bash
uh="/etc/usr_hosts"
sudo cp ./conf/*.conf /etc/
sudo sh -c "echo '# usr hosts' > ${uh}"
sudo chmod 777 ${uh}
sudo /etc/init.d/dnsmasq restart
