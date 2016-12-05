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
)


var (
	WorkPath = GetWorkPath()
	GetConfig = goini.SetConfig(WorkPath + "config.conf")
	corpid = GetConfig.GetValue("weixin", "corpid")
	key = GetConfig.GetValue("weixin", "key")
	secret = GetConfig.GetValue("weixin", "secret")
)


func main(){
	web := martini.Classic()
	web.Use(
		render.Renderer(render.Options{
			IndentJSON: true,
			Charset: "UTF-8",
		}),
	)

	web.Post("/wxauth", WxAuth)
	web.Post("/sendmsg", SendMsg)

	if port := GetConfig.GetValue("http", "port"); port == "no value" {
		fmt.Println("获取配置文件Http服务端口服务,使用默认4567!")
		WriteLog("获取配置文件Http服务端口服务,使用默认4567!")
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
	ToUser string `json:"touser"`
	MsgType string `json:"msgtype"`
	AgentID int `json:"agentid"`
	Text Content `json:"text"`
}

var Token AccessToken

func SendMsg(req *http.Request, ren render.Render){
	toUser := req.PostFormValue("tos")
	content := req.PostFormValue("content")

	newContent := Content{
		Content:content,
	}

	newMsgPost := MsgPost{
		ToUser:toUser,
		MsgType:"text",
		AgentID:0,
		Text:newContent,
	}

	var err error
	if time.Now().Unix() > Token.TimeOut{
		Token, err = GetAccessTokenFromWeixin()
		if err != nil{
			ren.Text(200,err.Error())
			return
		}
	}

	url := "https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token=" + Token.AccessToken
	result, err := Post(url, newMsgPost)
	WriteLog("发送信息给", toUser,string(result),"信息内容:\n", content)
	ren.Text(200, string(result))

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


//开启回调模式验证
func WxAuth(req *http.Request, ren render.Render){
	body, err := ioutil.ReadAll(req.Body)
	ren.Data(200, body)
	WriteLog(string(body), err)

	req.ParseForm()
	echostr := req.Form["echostr"][0]
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
	if string(id) == corpid{
		fmt.Println(string(x[20:20 + length]))
		ren.Data(200, x[20:20 + length])
	}else {
		WriteLog("微信验证appID错误!")
	}
	return
}



type AccessToken struct {
	AccessToken string `json:"access_token"`
	ExpiresIn int	`json:"expires_in"`
	TimeOut int64
}

//从微信获取 AccessToken
func GetAccessTokenFromWeixin()(newAccess AccessToken, err error){
	WxAccessTokenUrl := "https://qyapi.weixin.qq.com/cgi-bin/gettoken?corpid=" + corpid + "&corpsecret=" + secret

	tr := &http.Transport{
		TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
		DisableCompression: true,
	}
	client := &http.Client{Transport: tr}
	result, _ := client.Get(WxAccessTokenUrl)
	res, err := ioutil.ReadAll(result.Body)

	fmt.Println(string(res))

	if err != nil{
		WriteLog("获取微信 Token 返回数据错误: ", err)
		return newAccess, err
	}
	err = json.Unmarshal(res, &newAccess)
	if err != nil{
		WriteLog("获取微信 Token 返回数据解析 Json 错误: ", err)
		newAccess.TimeOut = time.Now().Unix() + int64(newAccess.ExpiresIn) - 1000
		return newAccess, err
	}

	return newAccess, err
}



//获取当前运行路径
func GetWorkPath() (string) {
	if dir, err := os.Getwd(); err == nil {
		return dir + "/"
	}
	return "./"
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
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}
