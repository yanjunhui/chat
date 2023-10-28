package crop

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

// Err 微信返回错误
type Err struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

//AccessToken 微信企业号请求Token
type AccessToken struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	Err
	ExpiresInTime time.Time
}

//Client 微信企业号应用配置信息
type Client struct {
	CropID      string
	AgentID     int
	AgentSecret string
	Token       AccessToken
}

//Result 发送消息返回结果
type Result struct {
	Err
	InvalidUser  string `json:"invaliduser"`
	InvalidParty string `json:"infvalidparty"`
	InvalidTag   string `json:"invalidtag"`
}

//Content 文本消息内容
type Content struct {
	Content string `json:"content"`
}

//Message 消息主体参数
type Message struct {
	ToUser  string  `json:"touser"`
	ToParty string  `json:"toparty"`
	ToTag   string  `json:"totag"`
	MsgType string  `json:"msgtype"`
	AgentID int     `json:"agentid"`
	Text    Content `json:"text"`
}

//New 实例化微信企业号应用
func New(cropID string, agentID int, AgentSecret string) *Client {

	c := new(Client)
	c.CropID = cropID
	c.AgentID = agentID
	c.AgentSecret = AgentSecret
	return c
}

//Send 发送信息
func (c *Client) Send(msg Message) error {

	c.GetAccessToken()

	msg.AgentID = c.AgentID
	url := "https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token=" + c.Token.AccessToken

	resultByte, err := JSONPost(url, msg)
	if err != nil {
		err = errors.New("请求微信接口失败: " + err.Error())
		log.Println(err)
		return err
	}

	result := Result{}
	err = json.Unmarshal(resultByte, &result)
	if err != nil {
		err = errors.New("解析微信接口返回数据失败: " + err.Error())
		log.Println(err)
		return err
	}

	if result.ErrCode != 0 {
		err = errors.New("发送消息失败: " + result.ErrMsg)
		log.Println(err)

	}

	if result.InvalidUser != "" || result.InvalidTag != "" || result.InvalidParty != "" {
		err = fmt.Errorf("消息发送成功, 但是有部分目标无法送达: %s%s%s", result.InvalidUser, result.InvalidParty, result.InvalidTag)
		log.Println(err)
	}

	return err
}

//GetAccessToken 获取回话token
func (c *Client) GetAccessToken() {
	var err error
	if c.Token.AccessToken == "" || c.Token.ExpiresInTime.Before(time.Now()) {
		c.Token, err = getAccessTokenFromWeixin(c.CropID, c.AgentSecret)
		if err != nil {
			log.Println("获取token失败: ", err)
			return
		}
		c.Token.ExpiresInTime = time.Now().Add(time.Duration(c.Token.ExpiresIn-1000) * time.Second)
	}
}

//从微信服务器获取token
func getAccessTokenFromWeixin(cropID, secret string) (TokenSession AccessToken, err error) {
	WxAccessTokenURL := "https://qyapi.weixin.qq.com/cgi-bin/gettoken?corpid=" + cropID + "&corpsecret=" + secret

	tr := &http.Transport{
		TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
		DisableCompression: true,
	}
	client := &http.Client{Transport: tr}
	result, err := client.Get(WxAccessTokenURL)
	if err != nil {
		return TokenSession, err
	}

	res, err := ioutil.ReadAll(result.Body)
	if err != nil {
		return TokenSession, err
	}

	defer result.Body.Close()

	err = json.Unmarshal(res, &TokenSession)
	if err != nil {
		return TokenSession, err
	}

	if TokenSession.ExpiresIn == 0 || TokenSession.AccessToken == "" {
		err = fmt.Errorf("获取微信错误代码: %v, 错误信息: %v", TokenSession.ErrCode, TokenSession.ErrMsg)
		return TokenSession, err
	}

	return TokenSession, err
}

//JSONPost Post请求json数据
func JSONPost(url string, data interface{}) ([]byte, error) {
	jsonBody, err := encodeJSON(data)
	if err != nil {
		return nil, err
	}
	r, err := http.Post(url, "application/json;charset=utf-8", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	return body, err
}

func encodeJSON(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
