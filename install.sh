#bash
wget https://github.com/Casxt/DDNS/archive/master.zip
unzip master.zip
rm -f master.zip
go get github.com/aliyun/alibaba-cloud-sdk-go/sdk
go build -o DDNS-master/DDNS DDNS-master/AliYun-DDNS.go 
sudo cp DDNS-master/DDNS /etc/bin/DDNS
sudo mkdir /etc/DDNS
sudo cp DDNS-master/config.template.json /etc/DDNS/config.json
sudo cp DDNS-master/ddns.service /etc/systemd/system/ddns.service
echo "1. edit /etc/DDNS/config.json 
      2. run 'sudo systemctl start ddns'"
