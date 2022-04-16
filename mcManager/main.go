package main

import (
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

func main() {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	//webInterface(router)
	//router.LoadHTMLGlob("HTML/*")
	log.Println("asd")
}
