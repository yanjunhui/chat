package main

import (
	"github.com/yanjunhui/chat/crop"
	"log"
)

//content := "[P0][OK][192.168.11.26_ofmon][][【critical】与主mysql同步延迟超过10s！ all(#3) seconds_behind_master port=3306 0>10][O1 2017-04-17 08:55:00]"

var (
	//微信企业号配置相关

	//corpID 微信企业号CropID
	corpID = "wwc84919127e378d9a"

	//EncodingAESKey 微信企业号加密Key
	EncodingAESKey = "P8uAU2eYOcCtXtrRCNv3iKxzc5HW6GcKI2Ri1slR3Ih"


	client *crop.Client
)

func init() {
	client = crop.New(corpID, 1000008, "Hw7Cw_xxqU78NPC9vfRhC0VQ9oBDtYO52W-4T8wt_3A")

}

func main() {
	msg := crop.Message{}
	msg.ToUser = "yanjunhui"
	msg.MsgType = "text"

	msg.Text = crop.Content{Content: "测试"}
	err := client.Send(msg)
	if err != nil {
		log.Println(err)
	}

	/*
	for i := 0; i <= 60 ; i++{
		msg.Text = crop.Content{Content: fmt.Sprintf("%d", i)}
		err := client.Send(msg)
		if err != nil {
			log.Println(err)
		}
	}
	*/
}

