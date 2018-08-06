package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
)

func main() {
	type Config struct {
		AccessKeyId     string
		AccessKeySecret string
		InterfaceName   string
	}
	var (
		netemp, net4, net6 *net.IPNet
		config             Config
	)
	b, err := ioutil.ReadFile("config.json")
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(b, &config); err != nil {
		panic(err)
	}

	//check config
	switch {
	case config.AccessKeyId == "":
		panic("invalid AccessKeyId")
	case config.AccessKeySecret == "":
		panic("invalid AccessKeySecret")
	case config.InterfaceName == "":
		panic("invalid InterfaceName")
	}

	//Get Local Ip Addr
	ifaces, err := net.Interfaces()
	if err != nil {
		panic(err)
	}

	for _, i := range ifaces {
		addrs, _ := i.Addrs()
		if i.Name != config.InterfaceName {
			continue
		}
		for _, addr := range addrs {
			netemp = addr.(*net.IPNet)
			if netemp.IP.To4() != nil {
				if netemp.IP.IsGlobalUnicast() && (net4 == nil || !net4.Contains(netemp.IP)) {
					net4 = netemp
				}
				// ip.IP.To16() != nil could not use to check ipv6! it will always success
			} else {
				if netemp.IP.IsGlobalUnicast() && (net6 == nil || !net6.Contains(netemp.IP)) {
					net6 = netemp
				}
			}
		}
	}
	if net4 != nil {
		fmt.Println("found local ipv4:", net4.String())
	}
	if net6 != nil {
		fmt.Println("found local ipv6:", net6.String())
	}
	if net4 == nil && net6 == nil {
		fmt.Println(ifaces)
		panic("localaddress not found")
	}

	client, err := sdk.NewClientWithAccessKey("cn-hangzhou", config.AccessKeyId, config.AccessKeySecret)
	if err != nil {
		panic(err)
	}

	request := requests.NewCommonRequest()
	request.Domain = "alidns.aliyuncs.com"
	request.Version = "2015-01-09"
	request.ApiName = "DescribeDomains"

	request.QueryParams["PageNumber"] = "1"
	request.QueryParams["PageSize"] = "1"

	response, err := client.ProcessCommonRequest(request)
	if err != nil {
		panic(err)
	}

	fmt.Println(response.GetHttpContentString())

}
