package controller

import (
	"bytes"
	"flag"
	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/keington/alertService/internel/biz/handler"
	"github.com/keington/alertService/internel/biz/models"
	"io"
	"log/slog"
	"net/http"
)

/**
 * @author: x.gallagher.anderson@gmail.com
 * @time: 2024/1/11 22:24
 * @file: alert_message_webhook_controller.go
 * @description: lark_webhook_router
 */

var hookUrl = flag.String("url", "https://localhost", "lark bot url")

const LarkRobotURL = "https://open.larksuite.com/open-apis/bot/v2/hook/27562c31-1810-4c08-b2ef-344ad2b99648"

// AlertMessageWebhookController 路由
func AlertMessageWebhookController(c *gin.Context) {
	var notification models.Notification

	err := c.ShouldBindJSON(&notification)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	slog.Info("received AlertManager alarm: ", notification)

	switch notification.Status {
	case "resolved":
		handleResolvedAlert(c, notification)
	case "firing":
		handleFiringAlert(c, notification)
	default:
		slog.Info("unknown alert status, skip sending message to lark server")
		c.JSON(http.StatusOK, gin.H{
			"message": "unknown alert status, skip sending message to lark server",
		})
	}
}

// handleResolvedAlert 处理告警恢复的情况
func handleResolvedAlert(c *gin.Context, notification models.Notification) {
	larkReq, err := handler.AlertResolvedTransformHandle(notification)
	if err != nil {
		slog.Error("[ERROR] failed to transform alertManager notification: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	slog.Info("alert has been resolved, skip sending message to lark server")

	sendMessageToLarkServer(c, larkReq)
}

// handleFiringAlert 处理告警触发的情况
func handleFiringAlert(c *gin.Context, notification models.Notification) {
	larkRequest, err := handler.AlertFiringTransformHandle(notification)
	if err != nil {
		// Handle the error
		slog.Error("failed to transform alertManager notification: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	slog.Info("alert has been resolved, skip sending message to lark server")
	sendMessageToLarkServer(c, larkRequest)
}

// sendMessageToLarkServer 发送消息到飞书机器人
func sendMessageToLarkServer(c *gin.Context, larkRequest *models.LarkRequest) {

	// var hook conf.Hook

	bytesData, _ := sonic.Marshal(larkRequest)
	req, _ := http.NewRequest("POST", *hookUrl, bytes.NewReader(bytesData))
	req.Header.Add("content-type", "application/json")
	res, err := http.DefaultClient.Do(req)

	if err != nil {
		slog.Error("request to lark server failed: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			// Handle the error
		}
	}(res.Body)

	body, _ := io.ReadAll(res.Body)
	var larkResponse models.LarkResponse
	err = sonic.Unmarshal(body, &larkResponse)

	if err != nil {
		slog.Error("failed to obtain response from lark server: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	slog.Info("successfully sent message to lark server")
	c.JSON(http.StatusOK, gin.H{
		"code":    larkResponse.Code,
		"message": larkResponse.Msg,
		"data":    larkResponse.Data,
	})
}
