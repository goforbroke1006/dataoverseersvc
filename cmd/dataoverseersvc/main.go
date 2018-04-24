package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	kitlog "github.com/go-kit/kit/log"
	"github.com/go-redis/redis"
	_ "github.com/lib/pq"

	"github.com/goforbroke1006/dataoverseersvc"
	"github.com/goforbroke1006/dataoverseersvc/config"
	"github.com/goforbroke1006/dataoverseersvc/mailing"
	"github.com/goforbroke1006/dataoverseersvc/validation"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU() / 2)

	validation.Register(&validation.InListValidator{})
	validation.Register(&validation.InRangeValidator{})
}

func main() {
	var (
		cfgFile    = flag.String("сfg-file", "config.yml", "-сfg-file=config.yml - configuration file")
		logFile    = flag.String("log-file", "", "-log-file=/var/log/yourFileName.log - log file location")
		tps        = flag.Uint("tps", 500, "-tps - average daemon load (desired number of transaction)")
		reportSize = flag.Uint("rsize", 5000, "-rsize - count of message in report")
	)
	flag.Parse()

	cfg, err := config.LoadConfig(*cfgFile)
	if nil != err {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	errCh := make(chan error)
	go interruptHandler(errCh)

	var logger kitlog.Logger
	if "" == *logFile {
		logger = kitlog.NewJSONLogger(os.Stdout)
	} else {
		fi, err := os.OpenFile(*logFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0777)
		if nil != err {
			panic(err)
		}
		defer fi.Close()
		logger = kitlog.NewJSONLogger(fi)
	}
	logger = kitlog.With(logger, "@timestamp", kitlog.DefaultTimestampUTC)
	logger = kitlog.With(logger, "@message", "info")
	logger = kitlog.With(logger, "caller", kitlog.DefaultCaller)
	logger.Log("msg", "hello")
	defer logger.Log("msg", "goodbye")

	connStr := fmt.Sprintf("%s://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.Connection.Driver,
		cfg.Connection.Username, cfg.Connection.Password,
		cfg.Connection.Host, cfg.Connection.Port, cfg.Connection.Name,
	)
	db, err := sql.Open(cfg.Connection.Driver, connStr)
	defer db.Close()
	if nil != err {
		logger.Log("err", err.Error())
		os.Exit(1)
	}

	var mailer *mailing.Mailer
	if cfg.Mailer.Type == "gmail" {
		mailer = mailing.NewGmailMailer(cfg.Mailer.Username, cfg.Mailer.Password)
	} else {
		if nil == cfg.Mailer.Host || nil == cfg.Mailer.Port {
			logger.Log("err", "failed to init mailer: you should define host and port")
		}
		mailer = mailing.NewMailer(*cfg.Mailer.Host, *cfg.Mailer.Port,
			cfg.Mailer.Username, cfg.Mailer.Password,
			cfg.Mailer.Username)
	}

	redisKV := redis.NewClient(&redis.Options{
		Addr:        fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password:    cfg.Redis.Password,
		DB:          0, // use default DB
		DialTimeout: 5 * time.Second,
	})
	_, err = redisKV.Ping().Result()
	defer redisKV.Close()
	if nil != err {
		logger.Log("err", err.Error())
		os.Exit(1)
	}

	svc := dataoverseersvc.NewDataOverseer(logger, db, redisKV, mailer)

	for _, task := range cfg.Tasks {
		validationHub := validation.ValidationHub{}
		for _, s := range task.Subjects {
			validationHub.Setup(s.Type, s.Columns, s.Params)
		}

		reports := make(chan string, (*reportSize)*100)

		mail := make(chan string, 100)
		go svc.CollectReport(reports, *reportSize, mail)
		go svc.SendReport(mail, cfg.AdminEmail)

		metrics := make(chan dataoverseersvc.SqlContent, *tps)
		lastId := int64(0)
		go func() {
			for {
				count, err := svc.LoadNextMetricsPortion(
					task.Query, *tps, &lastId, task.FieldId, metrics)
				if nil != err {
					logger.Log("err", err.Error())
				}
				if count > 0 {
					logger.Log("msg", fmt.Sprintf("load new %d rows", count))
				}
			}
		}()
		go func() {
			semaphore := make(chan bool, runtime.NumCPU())
			for cnt := range metrics {
				semaphore <- true
				go func() {
					defer func() { <-semaphore }()
					svc.ValidateData(validationHub, cnt, task.KVField, reports)
				}()
			}
		}()
	}

	logger.Log("exit", <-errCh)
}

func interruptHandler(errCh chan error) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	errCh <- fmt.Errorf("%s", <-c)
}
