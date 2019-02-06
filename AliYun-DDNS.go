package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
)

type Config struct {
	// aliyun 账户密钥
	AccessKeyId     string
	AccessKeySecret string
	// 网卡名
	InterfaceName string
	// 域名
	Domain string
	// 轮询周期 Second
	Sleep int64
}

var (
	config                 *Config
	configPath, ipv4, ipv6 string
)

func main() {

	flag.StringVar(&(configPath), "c", "", "config path")
	flag.StringVar(&(configPath), "config", "", "config path")

	flag.StringVar(&(config.InterfaceName), "i", "", "use ip of a interface instead of get ip from public api")
	flag.StringVar(&(config.InterfaceName), "interface", "", "use ip of a interface instead of get ip from public api")

	flag.StringVar(&(config.Domain), "d", "", "domain name for ddns")
	flag.StringVar(&(config.Domain), "domain", "", "domain name for ddns")

	flag.Int64Var(&(config.Sleep), "t", 60*60, "how many seconds between per check")
	flag.Int64Var(&(config.Sleep), "time", 60*60, "how many seconds between per check")

	flag.StringVar(&(config.AccessKeyId), "key", "", "Aliyun AccessKeyId")

	flag.StringVar(&(config.AccessKeySecret), "sec", "", "Aliyun AccessKeySecret")

	if configPath != "" {
		config = loadConfig(configPath)
	}

	//check config
	if config.AccessKeyId == "" {
		panic("invalid AccessKeyId")
	}
	if config.AccessKeySecret == "" {
		panic("invalid AccessKeySecret")
	}

	/*
		if config.InterfaceName == "" {
			panic("invalid InterfaceName")
		}
	*/

	rpKeyWord, domainName := SpliteDomain(config.Domain)
	// loop to refresh
	for {
		if config.InterfaceName != "" {
			ipv4, ipv6 = getIpFromInterface(config.InterfaceName)
		} else {
			ipv4, ipv6 = getIpFromSBAPI()
		}
		updateRecoder(config.AccessKeyId, config.AccessKeySecret, rpKeyWord, domainName, ipv4, ipv6)
		time.Sleep(time.Duration(config.Sleep) * time.Second)
	}
}

//SpliteDomain into DomainName, RRKeyWord
//input a.b.g.com return a.b, g.com
func SpliteDomain(domain string) (DomainName, RRKeyWord string) {
	i := strings.LastIndex(domain[0:strings.LastIndex(domain, ".")], ".")
	return domain[0:i], domain[i+1 : len(domain)]
}

func loadConfig(configPath string) *Config {
	config = new(Config)
	b, err := ioutil.ReadFile(configPath)
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(b, config); err != nil {
		panic(err)
	}
	return config
}

func getIpFromSBAPI() (ipv4, ipv6 string) {
	resp, err := http.Get("https://api-ipv4.ip.sb/ip")
	if err != nil {
		ipv4 = ""
	} else {
		if body, err := ioutil.ReadAll(resp.Body); err != nil {
			ipv4 = string(body)
		}
	}

	resp, err = http.Get("https://api-ipv6.ip.sb/ip")
	if err != nil {
		ipv6 = ""
	} else {
		if body, err := ioutil.ReadAll(resp.Body); err != nil {
			ipv6 = string(body)
		}
	}
	return ipv4, ipv6
}

func getIpFromInterface(interfaceName string) (ipv4, ipv6 string) {
	var (
		netemp, net4, net6 *net.IPNet
	)

	//Get Local Ip Addr
	ifaces, err := net.Interfaces()
	if err != nil {
		panic(err)
	}

	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			fmt.Println("Fetch Address Failed:", err.Error())
			return
		}
		if i.Name != interfaceName {
			continue
		}
		for _, addr := range addrs {
			netemp = addr.(*net.IPNet)
			if netemp.IP.To4() != nil {
				if netemp.IP.IsGlobalUnicast() && (net4 == nil || !net4.Contains(netemp.IP)) {
					net4 = netemp
				}
			} else {
				if netemp.IP.IsGlobalUnicast() && (net6 == nil || !net6.Contains(netemp.IP)) {
					net6 = netemp
				}
			}
		}
	}

	if net4 == nil && net6 == nil {
		fmt.Println(ifaces)
		fmt.Println("Get Address Failed: InterFace Not exist or not have any ip")
		return "", ""
	}
	if net4 != nil {
		fmt.Println("found ipv4:", net4.String())
		ipv4 = net4.IP.String()
	}
	if net6 != nil {
		fmt.Println("found ipv6:", net6.String())
		ipv6 = net6.IP.String()
	}

	return ipv4, ipv6
}

func updateRecoder(AccessKeyId, AccessKeySecret, RRKeyWord, DomainName, ipv4, ipv6 string) {
	var (
		ipv4RecoderID, ipv6RecoderID string
		res                          interface{}
	)

	// 查询记录
	client, err := sdk.NewClientWithAccessKey("cn-hangzhou", AccessKeyId, AccessKeySecret)
	if err != nil {
		fmt.Println("Create Client Failed:", err.Error())
		return
	}

	request := requests.NewCommonRequest()
	request.Domain = "alidns.aliyuncs.com"
	request.Version = "2015-01-09"
	request.ApiName = "DescribeDomainRecords"
	request.QueryParams["DomainName"] = DomainName
	request.QueryParams["RRKeyWord"] = RRKeyWord
	request.QueryParams["PageNumber"] = "1"
	request.QueryParams["PageSize"] = "100"

	response, err := client.ProcessCommonRequest(request)
	switch {
	case err != nil:
		fmt.Println("Request Failed:", err.Error())
		return
	case !response.IsSuccess():
		fmt.Println("Get Recoder Log Failed", response.GetHttpContentString())
		return
	}

	if err := json.Unmarshal(response.GetHttpContentBytes(), &res); err != nil {
		fmt.Println("Unmarshal Api response Failed")
		return
	}

	TotalCount, ok := res.(map[string]interface{})["TotalCount"].(float64)
	if !ok {
		fmt.Println(reflect.TypeOf(res.(map[string]interface{})["TotalCount"]))
		fmt.Println("TotalCount Err, TotalCount:", res.(map[string]interface{})["TotalCount"], "; Type:", reflect.TypeOf(res.(map[string]interface{})["TotalCount"]))
		return
	}

	if TotalCount >= 1 { //.(int) >= 1
		for _, r := range res.(map[string]interface{})["DomainRecords"].(map[string]interface{})["Record"].([]interface{}) {
			recoder := r.(map[string]interface{})
			if recoder["RR"].(string) == RRKeyWord {
				switch recoder["Type"].(string) {
				case "A":
					if ipv4 != "" && ipv4 != recoder["Value"].(string) {
						ipv4RecoderID = recoder["RecordId"].(string)
						fmt.Println("Get Ipv4 recoder ID:", ipv4RecoderID, "; Value:", recoder["Value"].(string))
					} else {
						ipv4 = ""
						fmt.Println("Ipv4 not change:", recoder["Value"].(string))
					}
				case "AAAA":
					if ipv6 != "" && ipv6 != recoder["Value"].(string) {
						ipv6RecoderID = recoder["RecordId"].(string)
						fmt.Println("Get Ipv6 recoder ID:", ipv6RecoderID, "; Value:", recoder["Value"].(string))
					} else {
						ipv6 = ""
						fmt.Println("Ipv6 not change:", recoder["Value"].(string))
					}
				}
			}
		}
	}

	// 在ipv4改变的情况下
	if ipv4 != "" {
		request := requests.NewCommonRequest()
		request.Domain = "alidns.aliyuncs.com"
		request.Version = "2015-01-09"
		request.QueryParams["RR"] = RRKeyWord
		request.QueryParams["Type"] = "A"
		request.QueryParams["Value"] = ipv4
		if ipv4RecoderID == "" {
			// 创建新记录
			fmt.Println("Create Ipv4 Recoder")
			request.ApiName = "AddDomainRecord"
			request.QueryParams["DomainName"] = DomainName

			response, err := client.ProcessCommonRequest(request)
			switch {
			case err != nil:
				fmt.Println("Create Ipv4 Recoder Failed:", err.Error())
			case !response.IsSuccess():
				fmt.Println("Create Ipv4 Recoder Failed", response.GetHttpContentString())
			default:
				fmt.Println("ipv4 Recoder update to:", ipv4)
			}
		} else {
			// 修改旧纪录
			request.ApiName = "UpdateDomainRecord"
			request.QueryParams["RecordId"] = ipv4RecoderID
			response, err := client.ProcessCommonRequest(request)
			switch {
			case err != nil:
				fmt.Println("Update Ipv4 Recoder Failed:", err.Error())
			case !response.IsSuccess():
				fmt.Println("Update Ipv4 Recoder Failed:", response.GetHttpContentString())
			default:
				fmt.Println("ipv4 Recoder update to:", ipv4)
			}
		}
	}

	// 在ipv6改变的情况下
	if ipv6 != "" {
		request := requests.NewCommonRequest()
		request.Domain = "alidns.aliyuncs.com"
		request.Version = "2015-01-09"
		request.QueryParams["RR"] = RRKeyWord
		request.QueryParams["Type"] = "AAAA"
		request.QueryParams["Value"] = ipv6
		if ipv6RecoderID == "" {
			fmt.Println("Create Ipv6 Recoder")
			request.ApiName = "AddDomainRecord"
			request.QueryParams["DomainName"] = DomainName
			response, err := client.ProcessCommonRequest(request)
			switch {
			case err != nil:
				fmt.Println("Update Ipv6 Recoder Log Failed:", err.Error())
			case !response.IsSuccess():
				fmt.Println("Create Ipv6 Recoder Log Failed", response.GetHttpContentString())
			default:
				fmt.Println("ipv6 Recoder update to:", ipv6)
			}
		} else {
			request.ApiName = "UpdateDomainRecord"
			request.QueryParams["RecordId"] = ipv6RecoderID
			response, err := client.ProcessCommonRequest(request)
			switch {
			case err != nil:
				fmt.Println("Update Ipv6 Recoder Log Failed:", err.Error())
			case !response.IsSuccess():
				fmt.Println("Update Ipv6 Recoder Log Failed", response.GetHttpContentString())
			default:
				fmt.Println("ipv6 Recoder update to:", ipv6)
			}
		}
	}
}
