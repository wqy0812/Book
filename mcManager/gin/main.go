package main

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"net/http"
)

func restartMC(c *gin.Context) {
	err, out := nasManager.GetAllNASVolumeInfo()
	if err != nil {
		returnJson, _ := json.Marshal(SimpleReturn{Success: false, Msg: err.Error()})
		c.String(http.StatusOK, string(returnJson))
	} else {
		returnJson, _ := json.Marshal(SimpleReturn{Success: true, Msg: out})
		c.String(http.StatusOK, string(returnJson))
	}
	return
}

func indexHTML(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", "flysnow_org")
}

func adminInterceptor(f gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.Request.Header.Get("uid")
		sessionID := c.Request.Header.Get("sessionID")

		sckStr := access.AdminSessionCheck(uid, sessionID, sessionCache)
		var sck access.LoginReturn
		json.Unmarshal([]byte(sckStr), &sck)

		if sck.IsLogin == true {
			f(c)
		} else {
			println("session check failed")
			c.String(http.StatusUnauthorized, sckStr)
			return
		}
	}
}
func webInterface(router *gin.Engine) {
	router.GET("/", indexHTML)
	router.Static("/static", "static")
	router.GET("/sessionCheck", sessionCheck)
	router.GET("/login", login)
	router.GET("/logout", logout)
	router.GET("/restartMC", adminInterceptor(restartMC))
}

func main() {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	//webInterface(router)
	//router.LoadHTMLGlob("HTML/*")
	log.Println("asd")
}
