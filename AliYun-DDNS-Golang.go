package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"reflect"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
)

type Config struct {
	AccessKeyId     string
	AccessKeySecret string
	InterfaceName   string
	DomainName      string
	RRKeyWord       string
	Sleep           int64
	Uint            string
}

var config *Config

func main() {
	for {
		config = new(Config)
		b, err := ioutil.ReadFile("config.json")
		if err != nil {
			panic(err)
		}
		if err := json.Unmarshal(b, config); err != nil {
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
		case config.DomainName == "":
			panic("invalid InterfaceName")
		case config.RRKeyWord == "":
			panic("invalid InterfaceName")
		}
		updateRecoder()
		switch config.Uint {
		case "Second":
			time.Sleep(time.Duration(config.Sleep) * time.Second)
		case "Minute":
			time.Sleep(time.Duration(config.Sleep) * time.Minute)
		case "Hour":
			time.Sleep(time.Duration(config.Sleep) * time.Hour)
		default:
			fmt.Println("Unsupport time Unit:", config.Uint, "Use Minute instead")
			time.Sleep(time.Duration(config.Sleep) * time.Minute)
		}
	}
}

func updateRecoder() {

	var (
		netemp, net4, net6           *net.IPNet
		ipv4RecoderID, ipv6RecoderID string
		res                          interface{}
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
		if i.Name != config.InterfaceName {
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
		fmt.Println("Get Address Failed:", err.Error())
		return
	}
	if net4 != nil {
		fmt.Println("found local ipv4:", net4.String())
	}
	if net6 != nil {
		fmt.Println("found local ipv6:", net6.String())
	}

	client, err := sdk.NewClientWithAccessKey("cn-hangzhou", config.AccessKeyId, config.AccessKeySecret)
	if err != nil {
		fmt.Println("Create Client Failed:", err.Error())
		return
	}

	request := requests.NewCommonRequest()
	request.Domain = "alidns.aliyuncs.com"
	request.Version = "2015-01-09"
	request.ApiName = "DescribeDomainRecords"
	request.QueryParams["DomainName"] = config.DomainName
	request.QueryParams["RRKeyWord"] = config.RRKeyWord
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

	json.Unmarshal(response.GetHttpContentBytes(), &res)

	TotalCount, ok := res.(map[string]interface{})["TotalCount"].(float64)
	if !ok {
		fmt.Println(reflect.TypeOf(res.(map[string]interface{})["TotalCount"]))
		fmt.Println("TotalCount Err, TotalCount:", res.(map[string]interface{})["TotalCount"], "; Type:", reflect.TypeOf(res.(map[string]interface{})["TotalCount"]))
		return
	}

	if TotalCount >= 1 { //.(int) >= 1
		for _, r := range res.(map[string]interface{})["DomainRecords"].(map[string]interface{})["Record"].([]interface{}) {
			recoder := r.(map[string]interface{})
			if recoder["RR"].(string) == config.RRKeyWord {
				switch recoder["Type"].(string) {
				case "A":
					if net4 != nil && net4.IP.String() != recoder["Value"].(string) {
						ipv4RecoderID = recoder["RecordId"].(string)
						fmt.Println("Get Ipv4 recoder ID:", ipv4RecoderID, "; Value:", recoder["Value"].(string))
					} else {
						net4 = nil
						fmt.Println("Ipv4 not change:", recoder["Value"].(string))
					}
				case "AAAA":
					if net6 != nil && net6.IP.String() != recoder["Value"].(string) {
						ipv6RecoderID = recoder["RecordId"].(string)
						fmt.Println("Get Ipv6 recoder ID:", ipv6RecoderID, "; Value:", recoder["Value"].(string))
					} else {
						net6 = nil
						fmt.Println("Ipv6 not change:", recoder["Value"].(string))
					}
				}
			}
		}
	}

	if net4 != nil {
		request := requests.NewCommonRequest()
		request.Domain = "alidns.aliyuncs.com"
		request.Version = "2015-01-09"
		request.QueryParams["RR"] = config.RRKeyWord
		request.QueryParams["Type"] = "A"
		request.QueryParams["Value"] = net4.IP.String()
		if ipv4RecoderID == "" {
			fmt.Println("Create Ipv4 Recoder")
			request.ApiName = "AddDomainRecord"
			request.QueryParams["DomainName"] = config.DomainName

			response, err := client.ProcessCommonRequest(request)
			switch {
			case err != nil:
				fmt.Println("Create Ipv4 Recoder Log Failed:", err.Error())
			case !response.IsSuccess():
				fmt.Println("Create Ipv4 Recoder Log Failed", response.GetHttpContentString())
			default:
				fmt.Println("ipv4 Recoder update to:", net4.String())
			}
		} else {
			request.ApiName = "UpdateDomainRecord"
			request.QueryParams["RecordId"] = ipv4RecoderID
			response, err := client.ProcessCommonRequest(request)
			switch {
			case err != nil:
				fmt.Println("Update Ipv4 Recoder Log Failed:", err.Error())
			case !response.IsSuccess():
				fmt.Println("Update Ipv4 Recoder Log Failed:", response.GetHttpContentString())
			default:
				fmt.Println("ipv4 Recoder update to:", net4.String())
			}
		}
	}
	if net6 != nil {
		request := requests.NewCommonRequest()
		request.Domain = "alidns.aliyuncs.com"
		request.Version = "2015-01-09"
		request.QueryParams["RR"] = config.RRKeyWord
		request.QueryParams["Type"] = "AAAA"
		request.QueryParams["Value"] = net6.IP.String()
		if ipv6RecoderID == "" {
			fmt.Println("Create Ipv6 Recoder")
			request.ApiName = "AddDomainRecord"
			request.QueryParams["DomainName"] = config.DomainName
			response, err := client.ProcessCommonRequest(request)
			switch {
			case err != nil:
				fmt.Println("Update Ipv6 Recoder Log Failed:", err.Error())
			case !response.IsSuccess():
				fmt.Println("Create Ipv6 Recoder Log Failed", response.GetHttpContentString())
			default:
				fmt.Println("ipv6 Recoder update to:", net6.String())
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
				fmt.Println("ipv6 Recoder update to:", net6.String())
			}
		}
	}
}
