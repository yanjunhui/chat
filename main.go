package main

import (
	"github.com/labstack/echo/v4"
	"github.com/yanjunhui/chat/crop"
	"log"
)

//content := "[P0][OK][192.168.11.26_ofmon][][【critical】与主mysql同步延迟超过10s！ all(#3) seconds_behind_master port=3306 0>10][O1 2017-04-17 08:55:00]"

var (
	//corpID 微信企业号CropID(仅示例, 修改成自己的 corpID)
	corpID = "wwc84919127e378d9a"

	//EncodingAESKey 微信企业号加密Key (仅示例, 修改成自己的 key)
	EncodingAESKey = "P8uAU2eYOcCtXtrRCNv3iKxzc5HW6GcKI2Ri1slR3Ih"

	client *crop.Client
)

func init() {
	client = crop.New(corpID, 1000008, "Hw7Cw_xxqU78NPC9vfRhC0VQ9oBDtYO52W-4T8wt_3A")
}

func main() {
	e := echo.New()
	e.GET("/send", MessageRequest)
	err := e.Start(":4567")
	if err != nil {
		log.Println("启动服务失败:", err)
	}
}

func MessageRequest(ctx echo.Context) error {
	msg := new(crop.Message)
	err := ctx.Bind(msg)
	if err != nil {
		return ctx.String(500, err.Error())
	}

	msg.Text = crop.Content{Content: "测试"}
	err = client.Send(*msg)
	if err != nil {
		log.Println(err)
		return ctx.String(500, err.Error())
	}

	return ctx.String(200, "ok")
}
