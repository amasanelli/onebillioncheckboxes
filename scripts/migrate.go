package main

import (
	"context"
	"fmt"
	"strconv"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

type envSchema struct {
	REDIS_ADDRESSES       []string          `env:"REDIS_ADDRESSES,required"`
	REDIS_ADDRESSES_REMAP map[string]string `env:"REDIS_ADDRESSES_REMAP" envKeyValSeparator:"|"`
}

const (
	ENV_FILE_PATH                string = "./.env"
	REDIS_CHECKS_KEY             string = "checks"
	REDIS_CHECKS_COUNT_KEY       string = "checks_count"
	REDIS_CHECKS_PER_KEY         uint32 = 100_000_000
	REDIS_POOL_SIZE              int    = 100
	REDIS_MAX_ACTIVE_CONNECTIONS int    = REDIS_POOL_SIZE
)

func main() {
	envData := &envSchema{}

	if err := godotenv.Load(ENV_FILE_PATH); err != nil {
		fmt.Println(err)
	}

	if err := env.Parse(envData); err != nil {
		panic(fmt.Errorf("error parsing env vars: %s", err.Error()))
	}

	rCli := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:          envData.REDIS_ADDRESSES,
		PoolSize:       REDIS_POOL_SIZE,
		MaxActiveConns: REDIS_MAX_ACTIVE_CONNECTIONS,
		NewClient: func(opt *redis.Options) *redis.Client {
			if len(envData.REDIS_ADDRESSES_REMAP) > 0 {
				opt.Addr = envData.REDIS_ADDRESSES_REMAP[opt.Addr]
			}
			return redis.NewClient(opt)
		},
	})

	if err := rCli.ForEachShard(context.Background(), func(ctx context.Context, shard *redis.Client) error {
		return shard.Ping(ctx).Err()
	}); err != nil {
		panic(err)
	}
	defer rCli.Close()

	strSliceCheckboxes, err := rCli.ZRangeByScore(context.Background(), REDIS_CHECKS_KEY, &redis.ZRangeBy{Min: "-inf", Max: "inf"}).Result()
	if err != nil {
		panic(err)
	}

	checkboxes := len(strSliceCheckboxes)

	if checkboxes == 0 {
		return
	}

	for i := 0; i < checkboxes; i++ {
		strCheckbox := strSliceCheckboxes[i]

		fmt.Println(strCheckbox, i+1, checkboxes)

		int64Checkbox, err := strconv.ParseInt(strCheckbox, 10, 64)
		if err != nil {
			panic(err)
		}
		uint32Checkbox := uint32(int64Checkbox)
		float64Checkbox := float64(uint32Checkbox)
		keyIndex := (uint32Checkbox - 1) / REDIS_CHECKS_PER_KEY
		key := fmt.Sprintf("%s_%d", REDIS_CHECKS_KEY, keyIndex)

		if err := rCli.ZAdd(context.Background(), key, redis.Z{Score: float64Checkbox, Member: strCheckbox}).Err(); err != nil {
			panic(err)
		}
	}

	fmt.Println("keys: ok")

	if err := rCli.Set(context.Background(), REDIS_CHECKS_COUNT_KEY, checkboxes, 0).Err(); err != nil {
		panic(err)
	}

	fmt.Println("counter: ok")

	if err := rCli.Del(context.Background(), REDIS_CHECKS_KEY).Err(); err != nil {
		panic(err)
	}

	fmt.Println("old key deletion: ok")
}
