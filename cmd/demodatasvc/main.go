package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	_ "github.com/lib/pq"

	"bitbucket.org/goforbroke1006/dataoverseersvc/config"
	"bitbucket.org/goforbroke1006/dataoverseersvc/repo"
	"bitbucket.org/goforbroke1006/dataoverseersvc/util/rand"
)

func main() {
	var (
		cfgFile = flag.String("сfg-file", "config.yml", "-log-file=/var/log/yourFileName.log")
	)
	flag.Parse()

	cfg, err := config.LoadConfig(*cfgFile)
	if nil != err {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	fmt.Println(cfg.Connection)

	//dbinfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
	//	DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME)
	connStr := fmt.Sprintf("%s://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.Connection.Driver,
		cfg.Connection.Username,
		cfg.Connection.Password,
		cfg.Connection.Host,
		cfg.Connection.Port,
		cfg.Connection.Name,
	)

	db, err := sql.Open(cfg.Connection.Driver, connStr)
	if nil != err {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	defer db.Close()

	var users []int
	var devices []int

	stmt, err := db.Prepare("INSERT INTO users(email, name) VALUES($1, $2) RETURNING id")
	if nil != err {
		log.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		email := util_rand.RandStringRunes(8) + "@" + util_rand.RandStringRunes(5) + "." + util_rand.RandStringRunes(3)
		row := db.QueryRow("SELECT * FROM users WHERE email = $1 LIMIT 1;", email)
		u := repo.User{}
		row.Scan(&u)
		if u.Id.Valid {
			continue
		}

		name := util_rand.RandStringRunes(8) + " " + util_rand.RandStringRunes(12)
		var lastId int
		err := stmt.QueryRow(email, name).Scan(&lastId)
		if err != nil {
			log.Fatal(err)
		}
		users = append(users, lastId)
		log.Println(fmt.Sprintf("Add user # %d", lastId))
	}

	deviceStmt, err := db.Prepare("INSERT INTO devices(name, user_id) VALUES($1, $2) RETURNING id")
	if nil != err {
		log.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		userId := users[rand.Int()%len(users)]
		deviceName := util_rand.RandStringRunes(6) +
			"-" + util_rand.RandStringRunes(6) +
			"-" + util_rand.RandStringRunes(6) +
			"-" + util_rand.RandStringRunes(14)
		var deviceId int
		err := deviceStmt.QueryRow(deviceName, userId).Scan(&deviceId)
		if err != nil {
			log.Fatal(err)
		}
		devices = append(devices, deviceId)
		log.Println(fmt.Sprintf("Add device # %d for user # %d", deviceId, userId))
	}

	metricStmt, err := db.Prepare("INSERT INTO device_metrics(" +
		"device_id, " +
		"metric_1, metric_2, metric_3, metric_4, metric_5, " +
		"local_time, server_time) " +
		"VALUES(" +
		"$1, " +
		"$2, $3, $4, $5, $6, " +
		"$7, $8" +
		")")
	for {
		deviceId := devices[rand.Int()%len(devices)]
		_, err := metricStmt.Exec(deviceId,
			rand.Int()%100+1,
			rand.Int()%100+1,
			rand.Int()%100+1,
			rand.Int()%100+1,
			rand.Int()%100+1,
			time.Now().UTC(),
			time.Now().UTC(),
		)
		if err != nil {
			log.Fatal(err)
		}
		time.Sleep(50 * time.Millisecond)
		log.Print(". ")
	}
}
