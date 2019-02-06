# DDNS
使用go语言编写，依赖AliyunApiSDk, 公网获取ip模型需要IP.SB api的支持
## Usage
### 配置文件方法
将config.template.json重命名为config.json，填写密钥，网卡名称，域名，检查周期即可。

执行 

`DDNS -c 配置文件路径`
### 命令行方法
1. 设置公网ip为 ddns ip

`DDNS -d ddns.example.com -key 阿里云key -sec 阿里云Secret`

2. 使用网卡eth0 ip为ddns ip

`DDNS -d 域名 -key 阿里云key -sec 阿里云Secret -i eth0`

3. 设置公网ip为ddns ip, 每小时检测

`DDNS -d 域名 -key 阿里云key -sec 阿里云Secret -i eth0 -t 3600`
