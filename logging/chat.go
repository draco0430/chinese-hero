package logging

import (
	"log"
	"os"
	"time"
)

var (
	ChatLogFile    *os.File
	AdminFile      *os.File
	GameLogFile    *os.File
	RemoveItemFile *os.File
	BlacksmithFile *os.File
	JangboFile     *os.File
	HTShopFile     *os.File
	LoginLogFile   *os.File
	GlobErr        error
	GlobLocation   *time.Location
)

func InitLogFiles() {
	ChatLogFile, GlobErr = os.OpenFile("chat_log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if GlobErr != nil {
		log.Println("ChatFile :", GlobErr)
		return
	}

	AdminFile, GlobErr = os.OpenFile("admin_log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if GlobErr != nil {
		log.Println("AdminLog :", GlobErr)
		return
	}
	GameLogFile, GlobErr = os.OpenFile("game_log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if GlobErr != nil {
		log.Println("GameLog :", GlobErr)
		return
	}

	RemoveItemFile, GlobErr = os.OpenFile("remove_item.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if GlobErr != nil {
		log.Println("RemoveItem :", GlobErr)
		return
	}

	BlacksmithFile, GlobErr = os.OpenFile("blacksmith.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if GlobErr != nil {
		log.Println("Blacksmith :", GlobErr)
		return
	}

	JangboFile, GlobErr = os.OpenFile("jangbo.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if GlobErr != nil {
		log.Println("Jangbo :", GlobErr)
		return
	}

	HTShopFile, GlobErr = os.OpenFile("htshop.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if GlobErr != nil {
		log.Println("HTShop :", GlobErr)
		return
	}

	LoginLogFile, GlobErr = os.OpenFile("logins.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if GlobErr != nil {
		log.Println("LoginLogFile :", GlobErr)
		return
	}

	GlobLocation, GlobErr = time.LoadLocation("Asia/Shanghai")
	if GlobErr != nil {
		log.Println("GlobLocation :", GlobErr)
		return
	}

}

func AddLogFile(logType int, msg string) {
	if logType == 0 {
		// admin
		AdminFile.WriteString("[-] " + time.Now().In(GlobLocation).Format("02.01.2006 15:04:05") + " - " + msg + "\n")
	}

	if logType == 1 {
		// user chat
		ChatLogFile.WriteString("[*] " + time.Now().In(GlobLocation).Format("02.01.2006 15:04:05") + " - " + msg + "\n")
	}

	if logType == 3 {
		// Game log
		GameLogFile.WriteString("[+] " + time.Now().In(GlobLocation).Format("02.01.2006 15:04:05") + " - " + msg + "\n")
	}

	if logType == 4 {
		// Blacksmith Log
		BlacksmithFile.WriteString("[+] " + time.Now().In(GlobLocation).Format("02.01.2006 15:04:05") + " - " + msg + "\n")
	}

	if logType == 5 {
		// RemoteItem Log
		RemoveItemFile.WriteString("[+] " + time.Now().In(GlobLocation).Format("02.01.2006 15:04:05") + " - " + msg + "\n")
	}

	if logType == 6 {
		// Jangbo Log
		JangboFile.WriteString("[+] " + time.Now().In(GlobLocation).Format("02.01.2006 15:04:05") + " - " + msg + "\n")
	}

	if logType == 7 {
		// HTShop Log
		HTShopFile.WriteString("[+] " + time.Now().In(GlobLocation).Format("02.01.2006 15:04:05") + " - " + msg + "\n")
	}

	if logType == 8 {
		LoginLogFile.WriteString("[+] " + time.Now().In(GlobLocation).Format("02.01.2006 15:04:05") + " - " + msg + "\n")
	}
}
