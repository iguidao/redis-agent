package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"redis-agent/src/httpapi"
	"redis-agent/src/logger"
	"regexp"
	"strconv"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/tencentyun/cos-go-sdk-v5"
)

const (
	AccessKey           = "xxxxx"
	AccessKeyID         = "xxxxx"
	EndpointPub         = "xxxxx"
	RedisManagerUrl     = "https://redis-manager.iguidao.com/redis-manager/public/v1/analysisrdb"
	DefalutDisplayCount = 20
)

func init() {
	logger.SetupLogger()
}

func postrdb(cosdir, ipaddr string) {
	postData := map[string]string{
		"rdbname":  cosdir,
		"serverip": ipaddr,
	}
	postDataStr, _ := json.Marshal(postData)
	// logger.Debug("feishurenwu create:", addurl, string(postDataStr), HeaderData)
	httpCode, httpResult := httpapi.PostJson(RedisManagerUrl, postDataStr, nil)
	if !httpCode {
		logger.Error("通知redis manager失败：", httpCode, httpResult)
	}
	// logger.Info("通知redis manager成功：", string(postDataStr), httpCode, httpResult)
}

func putcos(filedir, cosdir string) bool {
	u, _ := url.Parse(EndpointPub)
	b := &cos.BaseURL{BucketURL: u}

	client := cos.NewClient(b, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  AccessKeyID,
			SecretKey: AccessKey,
		},
	})
	result, err := client.Object.Head(context.Background(), cosdir, nil)
	if err != nil && result.Response.StatusCode != 404 {
		logger.Error("kfs put fetch current data error", err.Error())
		return false
	}
	if result.Response.StatusCode == 200 {
		cosdir = fmt.Sprintf("%s.%d", cosdir, time.Now().Unix())
		logger.Info("File exists on cos so use another name: ", cosdir)
	}
	for i := 0; i < 5; i++ {
		f, err := os.Open(filedir)
		if err != nil {
			logger.Error("start to upload error: %v", cosdir, err.Error())
			return false
		}

		opt := &cos.ObjectPutOptions{
			ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{
				ContentType: "text/html",
			},
			ACLHeaderOptions: &cos.ACLHeaderOptions{
				// 如果不是必要操作，建议上传文件时不要给单个文件设置权限，避免达到限制。若不设置默认继承桶的权限。
				//XCosACL: "private",
			},
		}
		//上传
		_, err = client.Object.Put(context.Background(), cosdir, f, opt)

		if err == nil {
			logger.Info(filedir + " upload success! Just run the following command where you want get the file:" + cosdir)
			return true
		} else {
			logger.Error(filedir+"upload error:", err.Error())
			return false
		}
	}
	return false
}
func checkbig(cmdtime int64, serverip string) (bool, string) {
	c, err := redis.Dial("tcp", serverip)
	if err != nil {
		logger.Error("conn redis failed,", err)
		return false, ""
	}
	defer c.Close()
	knowtime, err := redis.Int64(c.Do("LASTSAVE"))
	if knowtime > cmdtime-60 {
		cfginfo, err := redis.String(c.Do("info", "server"))
		if err != nil {
			logger.Error("执行info命令失败：", err)
			return false, ""
		}
		re := regexp.MustCompile("/data/(.*)/redis.conf")
		matcharr := re.FindStringSubmatch(cfginfo)
		filedir := "/data/" + matcharr[len(matcharr)-1] + "/data/dump.rdb"
		logger.Info("成功拿到地址备份文件地址：", filedir)
		return true, filedir
	}
	return false, ""
}

func main() {
	var ipaddr string
	knowtime := time.Now().Unix()
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		logger.Error("获取本地IP地址失败： ", err)
		return
	}
	for _, address := range addrs {
		// 检查ip地址判断是否回环地址
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ipaddr = ipnet.IP.String()
			}
		}
	}
	// logger.Info("成功获取IP地址：", ipaddr)
	ok, filedir := checkbig(knowtime, ipaddr+":10001")
	if ok {
		cosdir := ipaddr + ":10001-" + strconv.FormatInt(knowtime, 10)
		if putcos(filedir, cosdir) {
			// logger.Info("putcos success.")
			postrdb(cosdir, ipaddr+":10001")
		}
	}
}
