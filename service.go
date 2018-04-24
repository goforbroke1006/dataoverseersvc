package dataoverseersvc

import (
	"database/sql"
	"fmt"

	"crypto/md5"
	"encoding/hex"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-redis/redis"
	"github.com/goforbroke1006/dataoverseersvc/mailing"
	"github.com/goforbroke1006/dataoverseersvc/repo"
	"github.com/goforbroke1006/dataoverseersvc/validation"
	"time"
)

type SqlContent map[string]interface{}

type DataOverseer interface {
	CollectReport(queue <-chan string, portionSize uint, out chan<- string)
	SendReport(report <-chan string, adminEmail string) error
	LoadNextMetricsPortion(query string, limit uint, lastID *int64, idFiledName string,
		queue chan<- SqlContent) (uint, error)
	ValidateData(rules validation.ValidationHub, c SqlContent, kvKey string, out chan<- string) error
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

func (svc dataOverseer) SendReport(mail <-chan string, adminEmail string) error {
	for {
		svc.mailer.Send(adminEmail, "Metrics mail", <-mail)
		svc.logger.Log("msg", "send errors mail to "+adminEmail)
	}
}

func (svc dataOverseer) LoadNextMetricsPortion(
	query string, limit uint, lastID *int64, idFiledName string, queue chan<- SqlContent,
) (count uint, err error) {
	stmt, err := svc.db.Prepare(query)
	rows, err := stmt.Query(lastID)
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

func (svc dataOverseer) ValidateData(rules validation.ValidationHub, c SqlContent, kvKey string, out chan<- string) error {
	alert := ""
	for column, value := range c {
		if ok, err := rules.Validate(column, value); !ok {
			out <- err.Error()
			alert = alert + err.Error() + " ; "
		}
	}
	if len(alert) > 0 {
		deviceID := c[kvKey].(int64)
		svc.StoreLastAlert(fmt.Sprintf("%d", deviceID), alert)
	}
	return nil
}

func (svc dataOverseer) StoreLastAlert(deviceId, message string) error {
	return svc.redis.Set(
		getMD5Hash(deviceId),
		"["+deviceId+"] "+message, 15*time.Minute,
	).Err()
}

func NewDataOverseer(
	logger kitlog.Logger,
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

func getMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}
