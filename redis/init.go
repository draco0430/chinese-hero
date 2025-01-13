package redis

import (
	"fmt"
	"time"

	"github.com/go-redis/redis"
)

var (
	client  *redis.Client
	Client2 *redis.Client
)

func InitRedis() error {

	/*
		redisHost := os.Getenv("REDIS_HOST")
		if redisHost == "" {
			return nil
		}

		redisPort := os.Getenv("REDIS_PORT")
		if redisPort == "" {
			log.Fatal("REDIS_PORT env variable is empty")
		}

		redisPassword := os.Getenv("REDIS_PASSWORD")

		redisScheme := os.Getenv("REDIS_SCHEME")
		if redisScheme == "" {
			redisScheme = "rediss"
		}
	*/
	var err error

	client = redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "111111",
		DB:       0,
	})
	_, err = client.Ping().Result()
	if err != nil {
		fmt.Println("Redis error")
	}

	Client2 = redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "111111",
		DB:       2,
	})
	_, err = Client2.Ping().Result()
	if err != nil {
		fmt.Println("Redis error")
	}

	return err
}

func Set(key string, value interface{}) error {
	return client.Set(key, value, time.Duration(0)).Err()
}
