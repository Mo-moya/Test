package main

import (
	"context"
	"github.com/tencent-connect/botgo"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/event"
	"github.com/tencent-connect/botgo/openapi"
	"github.com/tencent-connect/botgo/token"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

var config Config
var api openapi.OpenAPI
var ctx context.Context

var recodeTable = make(map[int]int)
var p = make(map[string]map[int]int)
var guildID string
var userID string

type Config struct {
	AppID uint64 `yaml:"appid"`
	Token string `yaml:"token"`
}

func init() {
	content, err := ioutil.ReadFile("./config.yaml")
	if err != nil {
		log.Println("读取配置文件出错， err = ", err)
		os.Exit(1)
	}

	err = yaml.Unmarshal(content, &config)
	if err != nil {
		log.Println("解析配置文件出错， err = ", err)
		os.Exit(1)
	}
	log.Println(config)
}

func punch(channelID string, messageID string) error {
	day := time.Now().Day()
	content := "请勿重复打卡"
	if recodeTable, ok := p[userID]; ok {
		if _, ok := recodeTable[day]; !ok {
			recodeTable[day] = 1
			content = "打卡成功"
		}
		p[userID] = recodeTable
	} else {
		m := make(map[int]int)
		m[day] = 1
		content = "打卡成功"
		p[userID] = m
	}

	_, err := api.PostMessage(ctx, channelID, &dto.MessageToCreate{
		MsgID: messageID, Content: content,
	})
	if err != nil {
		return err
	}
	return nil
}

func queryList(channelID string, messageID string) error {
	var content string
	day := time.Now().Day()
	for i := 1; i <= day; i++ {
		recoded := "已打卡"
		recodeTable = p[userID]
		if _, ok := recodeTable[i]; !ok {
			recoded = "未打卡"
		}
		tempDay := strconv.FormatInt(int64(i), 10)
		if i < 10 {
			tempDay = "0" + strconv.FormatInt(int64(i), 10)
		}
		key := "202205" + tempDay
		content = content + "\n" + strconv.FormatInt(int64(i), 10) + "、" + key + ":" + recoded
	}
	_, err := api.PostMessage(ctx, channelID, &dto.MessageToCreate{
		MsgID: messageID, Content: content,
	})
	if err != nil {
		return err
	}
	return nil
}

//atMessageEventHandler 处理 @机器人 的消息
func atMessageEventHandler(event *dto.WSPayload, data *dto.WSATMessageData) error {

	userID = data.Author.ID
	botMessage := data.Content

	log.Println("botMessage：" + botMessage)

	botMessage = strings.TrimRight(botMessage, " ")
	if strings.HasSuffix(botMessage, "打卡") {
		err := punch(data.ChannelID, data.ID)
		if err != nil {
			return err
		}
		return nil
	}

	if strings.HasSuffix(botMessage, "查询打卡信息") {
		err := queryList(data.ChannelID, data.ID)
		if err != nil {
			return err
		}
		return nil
	}

	if strings.HasSuffix(data.Content, "你好") {
		_, err := api.PostMessage(ctx, data.ChannelID, &dto.MessageToCreate{MsgID: data.ID, Content: "您好"})
		if err != nil {
			return err
		}
		return nil
	}

	// 如果指令都未被捕获，则说明输入指令有问题
	err := tryException(data.ChannelID, data.ID)
	if err != nil {
		return err
	}
	return nil
}

func tryException(channelID string, messageID string) error {
	_, err := api.PostMessage(ctx, channelID, &dto.MessageToCreate{
		MsgID: messageID, Content: "指令输入非法，请重新输入指令！",
	})
	if err != nil {
		return err
	}
	return nil
}

func main() {
	token := token.BotToken(config.AppID, config.Token)
	//获取沙箱的api
	//api = botgo.NewOpenAPI(token).WithTimeout(3 * time.Second)
	api = botgo.NewSandboxOpenAPI(token).WithTimeout(3 * time.Second)
	ctx = context.Background()
	ws, err := api.WS(ctx, nil, "")
	if err != nil {
		log.Fatalln("websocket错误， err = ", err)
		os.Exit(1)
	}

	var atMessage event.ATMessageEventHandler = atMessageEventHandler
	/*	var atMessage event.ATMessageEventHandler = func(event *dto.WSPayload, data *dto.WSATMessageData) error {
		fmt.Println("event:", event, "\ndata:", data)
		punch(data.ChannelID, data.ID)
		return nil
	}*/
	intent := event.RegisterHandlers(atMessage)         // 注册socket消息处理
	botgo.NewSessionManager().Start(ws, token, &intent) // 启动socket监听
}
