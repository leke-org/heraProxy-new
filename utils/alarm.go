package util

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var alarmHttpClient *http.Client

const (
	TOKEN  = "b15453caf08509ae62ce712b5e130fc789534cb7b97c3d70b30bba181f0c510f"
	SECRET = "SEC87f7a057bae0b44b78e848f33551f2e981e0a1d6555025746b31bab67b5c51f7"
)

type DingTalkMessage struct {
	At struct {
		AtMobiles []string `json:"atMobiles"`
		AtUserIds []string `json:"atUserIds"`
		IsAtAll   bool     `json:"isAtAll"`
	} `json:"at"`
	Text struct {
		Content string `json:"content"`
	} `json:"text"`
	MsgType string `json:"msgtype"`
}

func init() {
	alarmHttpClient = &http.Client{}
}

func SendAlarmToDingTalk(content string) {
	ts, sign := makeSignature()
	rawUrl := fmt.Sprintf("https://oapi.dingtalk.com/robot/send?access_token=%s&timestamp=%s&sign=%s", TOKEN, ts, sign)

	message := DingTalkMessage{
		At: struct {
			AtMobiles []string `json:"atMobiles"`
			AtUserIds []string `json:"atUserIds"`
			IsAtAll   bool     `json:"isAtAll"`
		}{
			IsAtAll: true,
		},
		Text: struct {
			Content string `json:"content"`
		}{
			Content: content,
		},
		MsgType: "text",
	}
	// 序列化JSON数据
	jsonValue, err := json.Marshal(message)
	if err != nil {
		fmt.Println(err)
		return
	}

	// 创建请求
	req, err := http.NewRequest("POST", rawUrl, bytes.NewBuffer(jsonValue))
	if err != nil {
		fmt.Println(err)
		return
	}
	// 设置请求头为application/json
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := alarmHttpClient.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	resp.Body.Close()
}

func makeSignature() (string, string) {
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)

	stringToSign := strings.Join([]string{timestamp, SECRET}, "\n")

	// 创建 HMAC-SHA256
	h := hmac.New(sha256.New, []byte(SECRET))
	h.Write([]byte(stringToSign))
	signData := h.Sum(nil)

	// Base64 编码
	sign := base64.StdEncoding.EncodeToString(signData)

	// URL 编码
	sign = url.QueryEscape(sign)

	return timestamp, sign
}
