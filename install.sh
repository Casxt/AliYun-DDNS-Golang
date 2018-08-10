#bash
curl -o ddns-src.zip https://github.com/Casxt/DDNS/archive/master.zip
unzip ddns-src.zip -d "ddns-src"
rm -f ddns-src.zip
go get github.com/aliyun/alibaba-cloud-sdk-go/sdk
go build ddns-src/AliYun-DDNS.go -o ddns-src/DDNS
sudo cp ddns-src/DDNS /etc/bin/DDNS
sudo mkdir /etc/DDNS
sudo cp ddns-src/config.template.json /etc/DDNS/config.json
sudo cp ddns-src/ddns.service /etc/systemd/system/ddns.service
echo "1. edit /etc/DDNS/config.json 
      2. run 'sudo systemctl start ddns'"
