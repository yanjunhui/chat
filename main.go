package main

import (
	"fmt"
	"os"
	"net/http"
	"time"
	"crypto/aes"
	"crypto/cipher"
	"bytes"
	"github.com/yanjunhui/goini"
	"encoding/base64"
	"encoding/binary"
	"crypto/tls"
	"io/ioutil"
	"encoding/json"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/render"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

var (
	WorkPath = GetWorkPath()
	GetConfig = goini.SetConfig(WorkPath + "config.conf")
	corpid = GetConfig.GetValue("weixin", "corpid")
	key = GetConfig.GetValue("weixin", "key")
	secret = GetConfig.GetValue("weixin", "secret")
)

func main() {
	web := martini.Classic()
	web.Use(
		render.Renderer(render.Options{
			IndentJSON: true,
			Charset: "UTF-8",
		}),
	)

	web.Get("/wxauth", WxAuth)
	web.Post("/sendmsg", SendMsg)

	if port := GetConfig.GetValue("http", "port"); port == "no value" {
		fmt.Println("获取配置文件Http服务端口服务失败,使用默认4567!")
		WriteLog("获取配置文件Http服务端口服务失败,使用默认4567!")
		web.RunOnAddr(":4567")
	} else {
		WriteLog("启动 Http 服务")
		web.RunOnAddr(":" + port)
	}
}


//发送信息
type Content struct {
	Content string `json:"content"`
}

type MsgPost struct {
	ToUser  string `json:"touser"`
	MsgType string `json:"msgtype"`
	AgentID int `json:"agentid"`
	Text    Content `json:"text"`
}

func SendMsg(req *http.Request, ren render.Render) {
	toUser := req.PostFormValue("tos")
	content := req.PostFormValue("content")

	info := ""
	x := strings.Split(content, "]")
	if len(x) > 1 {
		for _, v := range x {
			y := strings.Split(v, "[")
			if len(y) > 1 {
				for _, c := range y {
					if c != "" {
						if info == "" {
							info += c
						} else {
							info = info + "\n" + c
						}
					}
				}
			}
		}
	}

	newContent := Content{
		Content:info,
	}

	userList := strings.Split(toUser,",")

	if len(userList) > 1 {
		toUser = strings.Join(userList, "|")
	}

	newMsgPost := MsgPost{
		ToUser:toUser,
		MsgType:"text",
		AgentID:0,
		Text:newContent,
	}

	t := GetConfig.GetValue("token", "token")
	if t == "no value"{
		token, err := GetAccessTokenFromWeixin()
		if err != nil {
			ren.Text(200, err.Error())
			return
		}
		t = token.AccessToken
		GetConfig.SetValue("token", "token", token.AccessToken)
		GetConfig.SetValue("token", "timeout", Int64ToString(token.TimeOut))
	}


	timeoutStr := GetConfig.GetValue("token", "timeout")
	timeout, err := StringToInt64(timeoutStr)
	if err != nil {
		WriteLog("timeout转换类型失败!")
		return
	}
	if time.Now().Unix() > timeout {
		token, err := GetAccessTokenFromWeixin()
		if err != nil {
			ren.Text(200, err.Error())
			return
		}
		t = token.AccessToken
		GetConfig.SetValue("token", "token", token.AccessToken)
		GetConfig.SetValue("token", "timeout", Int64ToString(token.TimeOut))
	}


	url := "https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token=" + t
	result, err := Post(url, newMsgPost)
	if err != nil {
		WriteLog("请求微信错误!", err)
		return
	}
	WriteLog("发送信息给", toUser, string(result), "信息内容:", content)
	ren.Text(200, string(result))
}

//开启回调模式验证
func WxAuth(req *http.Request, ren render.Render) {
	req.ParseForm()
	echostr := req.FormValue("echostr")
	if echostr == ""{
		ren.Text(200,"无法获取请求参数, 请使用微信请求接口!")
		return
	}
	wByte, _ := base64.StdEncoding.DecodeString(echostr)
	key, _ := base64.StdEncoding.DecodeString(key + "=")
	keyByte := []byte(key)
	x, _ := AesDecrypt(wByte, keyByte)

	buf := bytes.NewBuffer(x[16:20])
	var length int32
	binary.Read(buf, binary.BigEndian, &length)

	//验证返回数据ID是否正确
	appIDstart := 20 + length
	id := x[appIDstart : int(appIDstart) + len(corpid)]
	if string(id) == corpid {
		WriteLog("微信微信的验证字符串为: ",string(x[20:20 + length]))
		ren.Data(200, x[20:20 + length])
	} else {
		WriteLog("微信验证appID错误!")
	}
	return
}

type AccessToken struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int        `json:"expires_in"`
	TimeOut     int64
}

//从微信获取 AccessToken
func GetAccessTokenFromWeixin() (newAccess AccessToken, err error) {

	WxAccessTokenUrl := "https://qyapi.weixin.qq.com/cgi-bin/gettoken?corpid=" + corpid + "&corpsecret=" + secret

	tr := &http.Transport{
		TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
		DisableCompression: true,
	}
	client := &http.Client{Transport: tr}
	result, _ := client.Get(WxAccessTokenUrl)
	res, err := ioutil.ReadAll(result.Body)

	if err != nil {
		WriteLog("获取微信 Token 返回数据错误: ", err)
		return newAccess, err
	}
	err = json.Unmarshal(res, &newAccess)
	if err != nil {
		WriteLog("获取微信 Token 返回数据解析 Json 错误: ", err)
		return newAccess, err
	}
	newAccess.TimeOut = time.Now().Unix() + int64(newAccess.ExpiresIn) - 1000
	return newAccess, err
}


//无参数Post请求数据
func Post(url string, data interface{}) (result []byte, err error) {
	b, err := json.Marshal(data)
	body := bytes.NewBuffer([]byte(b))
	//请求日志写入
	WriteLog("Post请求内容: " + body.String())
	tr := &http.Transport{
		TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
		DisableCompression: true,
	}
	client := &http.Client{Transport: tr, Timeout:10 * time.Second}
	if res, err := client.Post(url, "application/json;charset=utf-8", body); err == nil {
		result, err = ioutil.ReadAll(res.Body)
		return result, err
	} else {
		WriteLog("Post 请求失败:", err)
		return result, err
	}
}


//获取当前运行路径
func GetWorkPath() (string) {
	if file, err := exec.LookPath(os.Args[0]); err == nil {
		return filepath.Dir(file) + "/"
	}
	return "./"
}

//int64 类型转 string
func Int64ToString(i int64) string {
	return strconv.FormatInt(i, 10)
}

//string 类型转 int64
func StringToInt64(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

//写入日志
func WriteLog(a ...interface{}) {
	t := time.Now().Format("2006年01月02日15点04分05秒")
	f, _ := os.OpenFile(WorkPath + "info.log", os.O_CREATE | os.O_APPEND | os.O_RDWR, 0660)
	defer f.Close()
	fmt.Fprintln(f, t, a)
}



//AES解密
func AesDecrypt(crypted, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	blockMode := cipher.NewCBCDecrypter(block, key[:blockSize])
	origData := make([]byte, len(crypted))
	blockMode.CryptBlocks(origData, crypted)
	origData = PKCS5UnPadding(origData)
	return origData, nil
}

func PKCS5UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length - 1])
	return origData[:(length - unpadding)]
}
