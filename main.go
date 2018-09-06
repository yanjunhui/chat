package main

import (
	"log"

	"regexp"

	"os"
	"os/exec"
	"path/filepath"

	"strconv"

	"strings"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/yanjunhui/chat/crop"
	"github.com/yanjunhui/goini"
)

//content := "[P0][OK][192.168.11.26_ofmon][][【critical】与主mysql同步延迟超过10s！ all(#3) seconds_behind_master port=3306 0>10][O1 2017-04-17 08:55:00]"

var (
	WorkPath       = GetWorkPath()
	GetConfig      = goini.SetConfig(WorkPath + "config.conf")
	corpId         = GetConfig.GetValue("weixin", "CorpID")
	agentId        = GetConfig.GetValue("weixin", "AgentId")
	secret         = GetConfig.GetValue("weixin", "Secret")
	EncodingAESKey = GetConfig.GetValue("weixin", "EncodingAESKey")
)

func main() {

	e := echo.New()
	e.Use(middleware.Logger())
	e.POST("/send", SendMsg)

	port := GetConfig.GetValue("http", "port")
	if port == "no value" {
		e.Logger.Fatal(e.Start("0.0.0.0:4567"))
	} else {
		e.Logger.Fatal(e.Start("0.0.0.0:" + port))
	}
}

func SendMsg(ctx echo.Context) error {
	toUser := ctx.FormValue("tos")
	content := ctx.FormValue("content")
	toUser = strings.Replace(toUser, ",", "|", -1)

	r := regexp.MustCompile(`(\[(.*?)])`)
	result := r.FindAllStringSubmatch(content, -1)

	text := ""
	if result != nil {
		contentList := []string{}
		for _, v := range result {
			if len(v) == 3 {
				if v[2] != "" {
					contentList = append(contentList, v[2])
				}
			}
		}
		text = strings.Join(contentList, "\n")
	} else {
		text = content
	}

	client := crop.New(corpId, EncodingAESKey, StringToInt(agentId), secret)

	msg := crop.Message{}
	msg.ToUser = toUser
	msg.MsgType = "text"
	msg.Text = crop.Content{text}

	log.Printf("发送告警信息: %s, 接收用户: %s", text, toUser)

	err := client.Send(msg)
	if err != nil {
		log.Println(err)
	}

	return ctx.String(200, "ok")
}

func GetWorkPath() string {
	if file, err := exec.LookPath(os.Args[0]); err == nil {
		return filepath.Dir(file) + "/"
	}
	return "./"
}

func StringToInt(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		log.Printf("agent 类型转换失败, 请检查配置文件中 agentid 配置是否为纯数字(%v)", err)
		return 0
	}
	return n
}
