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

	"github.com/goforbroke1006/dataoverseersvc/config"
	"github.com/goforbroke1006/dataoverseersvc/mailing"
	"github.com/goforbroke1006/dataoverseersvc/validation"
	"github.com/goforbroke1006/dataoverseersvc"
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
	/*
		ctx := context.Background()
		ctx = context.WithValue(ctx, "tps", tps)
		ctx = context.WithValue(ctx, "rsize", reportSize)
		ctx = context.WithValue(ctx, "cfg", cfg)
		ctx = context.WithValue(ctx, "logger", logger)
		ctx = context.WithValue(ctx, "db", db)
		ctx = context.WithValue(ctx, "mailer", mailer)
		ctx = context.WithValue(ctx, "redis", redisKV)

		for _, task := range cfg.Tasks {
			go runTask(ctx, task)
		}*/

	svc := dataoverseersvc.NewDataOverseer(&logger, db, redisKV, mailer)

	for _, task := range cfg.Tasks {
		validationHub := validation.ValidationHub{}
		for _, s := range task.Subjects {
			validationHub.Setup(s.Type, s.Columns, s.Params)
		}

		reports := make(chan string, *tps)

		reportMsg := make(chan string, (*tps)*10)
		go svc.CollectReport(reports, *reportSize, reportMsg)
		go svc.SendReport(reportMsg, cfg.AdminEmail)

		metrics := make(chan dataoverseersvc.SqlContent, *tps)
		lastId := int64(0)
		go func() {
			for {
				count, err := svc.LoadNextMetricsPortion(
					task.Query, *tps, &lastId, task.FieldId, metrics)
				if nil!=err {
					logger.Log("err", err.Error())
				}
				logger.Log("msg", fmt.Sprintf("load new %d rows", count))
			}
		}()
		go func() {
			semaphore := make(chan bool, runtime.NumCPU())
			for cnt := range metrics {
				semaphore <- true
				go func() {
					defer func() { <-semaphore }()
					svc.ValidateData(validationHub, cnt, reports)
				}()
			}
		}()
	}

	logger.Log("exit", <-errCh)
}

/*
func runTask(ctx context.Context, task config.Task) {
	logger := ctx.Value("logger").(kitlog.Logger)
	db := ctx.Value("db").(*sql.DB)
	tps := ctx.Value("tps").(*uint)

	reports := make(chan string, 20)
	go sendReport(ctx, reports)

	validationHub := validation.ValidationHub{}
	var last = new(int64)
	*last = 0

	stmt, err := db.Prepare(task.Query)
	if nil != err {
		log.Fatal(err)
	}
	for _, s := range task.Subjects {
		validationHub.Setup(s.Type, s.Columns, s.Params)
	}
	ctx = context.WithValue(ctx, "validationHub", validationHub)
	ctx = context.WithValue(ctx, "task", task)

	rows := make(chan dataoverseersvc.SqlContent, *tps)
	go func() {
		for {
			c, err := findRowsForTask(ctx, rows, stmt, task.FieldId, last)
			if nil != err {
				logger.Log("err", err.Error())
			} else if c > 0 {
				logger.Log("msg", fmt.Sprintf("found rows %d", c))
			}
		}
	}()

	maxStreams := 200
	semaphore := make(chan bool, maxStreams)
	for {
		semaphore <- true
		go validateRow(semaphore, ctx, reports, <-rows)
	}
	for i := 0; i < cap(semaphore); i++ {
		semaphore <- true
	}
}

func sendReport(ctx context.Context, rc chan string) {
	rsize := ctx.Value("rsize").(*int)
	cfg := ctx.Value("cfg").(*config.Configuration)
	mailer := ctx.Value("mailer").(*mailing.Mailer)

	for {
		var counter = 0
		msg := ""
		for errMsg := range rc {
			msg = msg + errMsg + "\n"
			counter++
			if counter >= *rsize {
				break
			}
		}
		mailer.Send(cfg.AdminEmail, "Metrics report", msg)
	}
}

func findRowsForTask(
	ctx context.Context,
	c chan dataoverseersvc.SqlContent,
	stmt *sql.Stmt,
	idField string,
	lastID *int64,
) (int, error) {
	logger := ctx.Value("logger").(kitlog.Logger)

	rows, err := stmt.Query(*lastID)
	if nil != err {
		return 0, err
	}
	defer rows.Close()
	columns, _ := rows.Columns()

	counter := 0
	for rows.Next() {
		values := make([]interface{}, len(columns))
		for idx := range columns {
			values[idx] = new(repo.MetalScanner)
		}
		err := rows.Scan(values...)
		if nil != err {
			logger.Log("err", err.Error())
		}

		res := dataoverseersvc.SqlContent{}
		for pos, idx := range columns {
			res[idx] = values[pos].(*repo.MetalScanner).Value
		}
		c <- res
		counter++

		*lastID = res[idField].(int64)
	}
	return counter, nil
}

func validateRow(sem <-chan bool, ctx context.Context, reports chan string, c dataoverseersvc.SqlContent) {
	defer func() { <-sem }()

	logger := ctx.Value("logger").(kitlog.Logger)
	validationHub := ctx.Value("validationHub").(validation.ValidationHub)
	task := ctx.Value("task").(config.Task)

	id := c[task.FieldId].(int64)

	for k, v := range c {
		if ok, err := validationHub.Validate(k, v); !ok {
			reports <- fmt.Sprintf("err - %s [%d]", err.Error(), id)
			logger.Log("err", err.Error(), "id", id)
		}
	}
}
*/

func interruptHandler(errCh chan error) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	errCh <- fmt.Errorf("%s", <-c)
}
