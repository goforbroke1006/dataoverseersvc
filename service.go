package dataoverseersvc

import (
	"database/sql"
	"fmt"

	kitlog "github.com/go-kit/kit/log"
	"github.com/go-redis/redis"
	"github.com/goforbroke1006/dataoverseersvc/mailing"
	"github.com/goforbroke1006/dataoverseersvc/validation"
	"github.com/goforbroke1006/dataoverseersvc/repo"
)

type SqlContent map[string]interface{}

type DataOverseer interface {
	CollectReport(queue <-chan string, portionSize uint, out chan<- string)
	SendReport(report <-chan string, adminEmail string) error
	LoadNextMetricsPortion(query string, limit uint, lastID *int64, idFiledName string,
		queue chan<- SqlContent) (uint, error)
	ValidateData(rules validation.ValidationHub, c SqlContent, out chan<- string) error
	StoreLastAlert(deviceId, message string) error
}

type dataOverseer struct {
	logger kitlog.Logger
	db     *sql.DB
	redis  *redis.Client
	mailer *mailing.Mailer
}

func (svc dataOverseer) CollectReport(queue <-chan string, portionSize uint, out chan<- string) {
	for {
		var counter = uint(0)
		msg := ""
		for errMsg := range queue {
			msg = msg + errMsg + "\n"
			counter++
			if counter >= portionSize {
				break
			}
		}
		svc.logger.Log("msg",
			fmt.Sprintf("errors report ready, length = %d", len(msg)))
		out <- msg
	}
}

func (svc dataOverseer) SendReport(report <-chan string, adminEmail string) error {
	for {
		svc.mailer.Send(adminEmail, "Metrics report", <-report)
		svc.logger.Log("msg", "send errors report to "+adminEmail)
	}
}

func (svc dataOverseer) LoadNextMetricsPortion(
	query string, limit uint, lastID *int64, idFiledName string, queue chan<- SqlContent,
) (count uint, err error) {
	stmt, err := svc.db.Prepare(query)
	rows, err := stmt.Query(*lastID)
	if nil != err {
		return 0, err
	}
	defer rows.Close()
	columns, _ := rows.Columns()
	for rows.Next() {
		values := make([]interface{}, len(columns))
		for idx := range columns {
			values[idx] = new(repo.MetalScanner)
		}
		err := rows.Scan(values...)
		if nil != err {
			return 0, err
		}

		res := SqlContent{}
		for pos, idx := range columns {
			res[idx] = values[pos].(*repo.MetalScanner).Value
		}
		queue <- res

		*lastID = res[idFiledName].(int64)
		count++
	}
	return count, nil
}

func (svc dataOverseer) ValidateData(rules validation.ValidationHub, c SqlContent, out chan<- string) error {
	for column, value := range c {
		if ok, err := rules.Validate(column, value); !ok {
			out <- err.Error()
		}
	}
	return nil
}

func (svc dataOverseer) StoreLastAlert(deviceId, message string) error {
	return nil
}

func NewDataOverseer(
	logger *kitlog.Logger,
	db *sql.DB,
	redis *redis.Client,
	mailer *mailing.Mailer,
) DataOverseer {
	return &dataOverseer{
		logger: logger,
		db:     db,
		redis:  redis,
		mailer: mailer,
	}
}
