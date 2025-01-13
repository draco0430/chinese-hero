package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	_ "strings"
	"time"

	_ "hero-server/factory"
	_ "hero-server/security"

	"hero-server/ai"
	"hero-server/config"
	"hero-server/database"
	"hero-server/logging"
	"hero-server/nats"
	"hero-server/web"

	"github.com/robfig/cron"

	//_ "net/http/pprof"

	_ "github.com/KimMachineGun/automemlimit"
)

//var logger = logging.Logger

func initDatabase() {
	for {
		err := database.InitDB()
		if err == nil {
			log.Printf("Connected to database...")
			return
		}
		log.Printf("Database connection error: %+v, waiting 30 sec...", err)
		time.Sleep(time.Duration(30) * time.Second)
	}
}

/*
func initRedis() {
	for {
		err := redis.InitRedis()
		if err != nil {
			log.Printf("Redis connection error: %+v, waiting 30 sec...", err)
			time.Sleep(time.Duration(30) * time.Second)
			continue
		}

		log.Printf("Connected to redis...")
		go logger.StartLogging()


			if redisHost := os.Getenv("REDIS_HOST"); redisHost != "" {
				log.Printf("Connected to redis...")
				go logger.StartLogging()
			}


		return
	}
}
*/

func startServer() {
	cfg := config.Default
	port := cfg.Server.Port

	listen, err := net.Listen("tcp4", ":"+strconv.Itoa(port))
	if err != nil {
		log.Fatalf("Socket listen port %d failed,%s", port, err)
		os.Exit(1)
	}
	defer listen.Close()
	log.Printf("Begin listen port: %d", port)

	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Fatalln(err)
			continue
		}

		//API Security Check Start
		/*
			parsedIP := strings.Split(conn.RemoteAddr().String(), ":")
			if parsedIP[0] == "" {
				conn.Close()
				continue
			}

			security.BannedIPsMutex.Lock()
			val, have := security.BannedIPs[parsedIP[0]]
			security.BannedIPsMutex.Unlock()
			if have {
				if val >= 10 {
					conn.Close()
					continue
				}
			}

			if !security.CheckPlayer(parsedIP[0]) {
				conn.Close()

				security.BannedIPsMutex.Lock()
				tmpVal, tmpHave := security.BannedIPs[parsedIP[0]]
				if tmpHave {
					security.BannedIPs[parsedIP[0]] = tmpVal + 1
				} else {
					security.BannedIPs[parsedIP[0]] = tmpVal + 1
				}
				security.BannedIPsMutex.Unlock()

				continue
			}
		*/
		// API Security Check Finish

		/*

					//fmt.Println(parsedIP[0])

			for b := range BanList {
				if parsedIP[0] == BanList[b] {
					conn.Close()
					continue
				}
			}

					parsedIP := strings.Split(conn.RemoteAddr().String(), ":")
			if parsedIP[0] == "" {
				conn.Close()
				continue
			}

				if security.RemoteAddrs[parsedIP[0]] >= 3 {
					fmt.Println("Multi Client: ", parsedIP[0])
					conn.Close()
					continue
				}



				connectionSize := security.RemoteAddrs[parsedIP[0]]
				security.RemoteAddrs[parsedIP[0]] = connectionSize + 1
		*/

		ws := database.Socket{Conn: conn, WriteChan: make(chan struct{}, 1)}
		go ws.Read()
		//go ws.WriteHandler()
	}
}

func cronHandler() {
	c := cron.New()
	c.AddFunc("0 0 0 * * *", func() {
		database.RefreshAIDs()
		database.RefreshYingYangKeys()
		//database.ResetDaily()
		database.ResetDailyCheckIn()
	})

	c.AddFunc("@every 1m", func() {
		beijingLoc, tmpErr := time.LoadLocation("Asia/Shanghai")
		if tmpErr != nil {
			fmt.Println("FAİL: Load Location err: ", tmpErr)
			return
		}

		tmpNow := time.Now().In(beijingLoc).Hour()
		tmpNowMin := time.Now().In(beijingLoc).Minute()

		// Sabah
		/*if tmpNow == 10 {
			if tmpNowMin == 0 {
				// Start non divine war
				database.StartWarAutoFunc(false)
			}

			/*
				if tmpNowMin == 30 {
					// Start divine war
					database.StartWarAutoFunc(true)
				}

		}*/

		// Akşam
		if tmpNow == 22 {
			if tmpNowMin == 0 {
				// Start non divine war
				database.StartWarAutoFunc(false)
			}

			/*
				if tmpNowMin == 30 {
					// Start divine war
					database.StartWarAutoFunc(true)
				}
			*/
		}

		if tmpNow == 23 {
			if tmpNowMin == 0 {
				//database.StartGoldenBasinWar()
			}
		}

	})

	c.Start()
}

/*
func reloadBans() {
	for {
		tmpFile, err := os.Open("ipban.txt")
		if err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}
		scanner := bufio.NewScanner(tmpFile)
		BanList = []string{}
		for scanner.Scan() {
			BanList = append(BanList, scanner.Text())
		}
		tmpFile.Close()
		time.Sleep(time.Minute * 1)
	}
}
*/

func main() {

	//debug.SetGCPercent(-1)
	//debug.SetMemoryLimit(math.MaxInt64)
	//go reloadBans()

	//initRedis()
	logging.InitLogFiles()
	defer func() {
		logging.GameLogFile.Close()
		logging.AdminFile.Close()
		logging.ChatLogFile.Close()
		logging.RemoveItemFile.Close()
		logging.BlacksmithFile.Close()
		logging.JangboFile.Close()
		logging.HTShopFile.Close()
		logging.LoginLogFile.Close()
	}()
	initDatabase()
	cronHandler()
	//go http.ListenAndServe(":7777", nil)
	go web.StartWebServer()

	ai.Init()
	go database.UnbanUsers()
	go database.FixDropAndExp() // Temple bug fix TODO

	s := nats.RunServer(nil)
	defer s.Shutdown()

	c, err := nats.ConnectSelf(nil)
	defer c.Close()

	if err != nil {
		log.Fatalln(err)
	}

	//go api.InitGRPC()

	startServer()
}
