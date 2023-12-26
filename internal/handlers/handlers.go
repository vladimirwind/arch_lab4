package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
	_ "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/sharding"
)

type Users struct {
	gorm.Model
	Id            int `gorm:"primary_key"`
	User_name     string
	User_surname  string
	User_login    string
	User_password string
}
type User struct {
	ID       int    `json:"id,omitempty" gorm:"id,primarykey"`
	Name     string `json:"name"`
	Surname  string `json:"surname"`
	Login    string `json:"login"`
	Password string `json:"password"`
}
type UserMask struct {
	Name    string `json:"name"`
	Surname string `json:"surname"`
}
type Report struct {
	Name   string `json:"name"`
	UserId int    `json:"id"`
}

var incr = 0
var db *gorm.DB
var SQLdb *sql.DB
var mapa = make(map[int][3]string)
var redis_conn *redis.Client

func Finder(pattern1 string, pattern2 string) ([]int64, error) {
	pattern1 = strings.Replace(pattern1, "*", ".", -1)
	pattern2 = strings.Replace(pattern2, "*", ".", -1)
	var err error
	var result []int64
	for key, value := range mapa {
		matched, err := regexp.MatchString(pattern1, value[0])
		if err != nil {
			return []int64{-1}, err
		}
		matched2, err := regexp.MatchString(pattern2, value[1])
		if err != nil {
			return []int64{-1}, err
		}
		if matched && matched2 {
			fmt.Printf("Key: %d, Value: %s\n", key, value)
			result = append(result, int64(key))
		}
	}
	if len(result) > 0 {
		return result, nil
	}
	return []int64{-1}, err
}
func LoginFinder(login string) ([]int64, error) {
	var result []int64
	for key, value := range mapa {
		if value[2] == login {
			result = append(result, int64(key))
		}
	}
	if len(result) > 0 {
		return result, nil
	} else {
		return nil, errors.New("login not found")
	}
}
func Algos(value any) (suffix string, err error) {

	uid := incr % 2
	incr += 1
	return fmt.Sprintf("_%02d", uid), nil
}
func ConnectDB() (*gorm.DB, error) {
	env := os.Getenv("ENVIRONMENT")
	var prod_db *sql.DB
	var err error
	if env == "test" {
		prod_db, err = sql.Open("mysql", "admin:mai@tcp(localhost:3306)/maidb")
		if err != nil {
			return nil, errors.New("could not connect to db")
		}
	} else {
		prod_db, err = sql.Open("mysql", "admin:mai@tcp(mariadb:3306)/maidb")
		if err != nil {
			return nil, errors.New("could not connect to db")
		}
	}
	SQLdb = prod_db
	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		Conn: prod_db,
	}), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	middleware := sharding.Register(sharding.Config{
		ShardingKey:         "id",
		NumberOfShards:      2,
		PrimaryKeyGenerator: sharding.PKSnowflake,
		ShardingAlgorithm:   Algos,
	}, "users")
	err = gormDB.Use(middleware)
	if err != nil {
		return nil, fmt.Errorf("error occured when conencting to DB: %w", err)
	}
	err = gormDB.AutoMigrate(&Users{})
	if err != nil {
		return nil, fmt.Errorf("error occured when conencting to DB: %w", err)
	}
	db = gormDB
	return gormDB, nil

}

func ConnectRedis() error {
	env := os.Getenv("ENVIRONMENT")
	var client *redis.Client
	if env == "test" {
		client = redis.NewClient(&redis.Options{
			Addr:     "localhost:6379",
			Password: "",
			DB:       0,
		})
	} else {
		client = redis.NewClient(&redis.Options{
			Addr:     "redis:6379",
			Password: "",
			DB:       0,
		})
	}

	pong, err := client.Ping().Result()
	if err != nil {
		return err
	}
	fmt.Println(pong)
	redis_conn = client
	return nil
}

// CreateUser godoc
//
//	@Summary		Creates a new user
//	@Description	create a new user
//	@Accept			json
//	@Produce		json
//
//	@Tags			mai lab API
//
//	@Param			user_data	body		User	true	"User Data"
//	@Success		200			{object}	string
//	@Failure		400			{object}	gin.H
//	@Failure		404			{object}	gin.H
//	@Failure		500			{object}	gin.H
//	@Router			/user/create [post]
func CreateUser(c *gin.Context) {
	jsonData, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error ": err.Error()})
		return
	}
	var usr User
	err = json.Unmarshal(jsonData, &usr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error ": err.Error()})
		return
	}
	if len(usr.Login) == 0 || len(usr.Password) == 0 || len(usr.Name) == 0 || len(usr.Surname) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error ": "not all fields are provided"})
		return
	}
	err_cr := db.Exec("INSERT INTO users(id,user_name,user_surname,user_login,user_password) VALUES(?,?,?,?,?) RETURNING id",
		usr.ID, usr.Name, usr.Surname, usr.Login, usr.Password)
	if err_cr.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error ": err_cr.Error.Error()})
		return
	}
	mapa[usr.ID] = [3]string{usr.Name, usr.Surname, usr.Login}
	err = SaveUserId(*redis_conn, &usr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error ": err.Error()})
		return
	}
	c.JSON(200, "user has been created")
}

// FindUserByLogin godoc
//
//	@Summary		Find User By Id
//	@Description	Find User By Id
//	@Accept			json
//	@Produce		json
//
//	@Tags			mai lab API
//
//	@Param			user_id	path		string	true	"User id"
//	@Success		200			{object}	[]UserMask
//	@Failure		400			{object}	gin.H
//	@Failure		404			{object}	gin.H
//	@Failure		500			{object}	gin.H
//	@Router			/user/findById/{id} [get]
func FindUserByID(c *gin.Context) {
	usr_id := c.Params.ByName("id")
	data := UserFromCacheId(*redis_conn, usr_id)
	var res User
	err := json.Unmarshal([]byte(data), &res)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error ": err.Error()})
		return
	}
	var res2 User
	var fromCache, fromDB bool
	if res.Name == "" {
		fromDB = true
		i, err := strconv.ParseInt(usr_id, 10, 64)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error ": err.Error()})
			return
		}
		errs := db.Raw("SELECT user_name, user_surname FROM users WHERE id = ?", i).Scan(&res2)
		if errs.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error ": errs.Error.Error()})
			return
		}
		if len(res2.Name) == 0 {
			errs := db.Raw("SELECT user_name, user_surname FROM users WHERE id = ?", i).Scan(&res2)
			if errs.Error != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error ": errs.Error.Error()})
				return
			}
		}
	} else {
		fromCache = true
	}
	if fromCache {
		c.JSON(200, gin.H{"found user(s) ": res})
		return
	}
	if fromDB {
		if len(res2.Name) == 0 {
			c.JSON(200, "User with such ID not found!")
			return
		}
		err = SaveUserId(*redis_conn, &res2)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error ": err.Error()})
			return
		}
		c.JSON(200, gin.H{"found user(s) ": res2})
	}

}

// FindUserByLogin godoc
//
//	@Summary		Find User By Login
//	@Description	Find User By Login
//	@Accept			json
//	@Produce		json
//
//	@Tags			mai lab API
//
//	@Param			user_log	path		string	true	"User Login"
//	@Success		200			{object}	[]UserMask
//	@Failure		400			{object}	gin.H
//	@Failure		404			{object}	gin.H
//	@Failure		500			{object}	gin.H
//	@Router			/user/findLogin/{user_log} [get]
func FindUserByLogin(c *gin.Context) {
	usr_log := c.Params.ByName("user_log")
	if len(usr_log) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error ": "login not provided"})
		return
	}
	data := UserFromCacheLogin(*redis_conn, usr_log)
	if len(data) > 0 {
		var res_user User
		err := json.Unmarshal([]byte(data), &res_user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error ": err.Error()})
			return
		}
		c.JSON(200, gin.H{"found user(s) ": res_user})
		return
	}
	type Result struct {
		User_name    string
		User_surname string
	}
	array, err := LoginFinder(usr_log)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error ": err.Error()})
		return
	}
	var result []Result
	for _, val := range array {
		var res Result
		errs := db.Raw("SELECT user_name, user_surname FROM users WHERE id = ?", val).Scan(&res)
		if errs.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error ": errs.Error.Error()})
			return
		}
		if len(res.User_name) == 0 {
			errs := db.Raw("SELECT user_name, user_surname FROM users WHERE id = ?", val).Scan(&res)
			if errs.Error != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error ": errs.Error.Error()})
				return
			}
		}
		result = append(result, res)
	}
	err = SaveUserLogin(*redis_conn, &User{Name: result[0].User_name, Surname: result[0].User_surname})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error ": err.Error()})
		return
	}
	c.JSON(200, gin.H{"found user(s) ": result})
}

// FindUserByMask godoc
//
//	@Summary		Find User By Mask
//	@Description	Find User By Mask
//	@Accept			json
//	@Produce		json
//
//	@Tags			mai lab API
//
//	@Param			user_log	body		UserMask	true	"User Data with mask"
//	@Success		200			{object}	[]UserMask
//	@Failure		400			{object}	gin.H
//	@Failure		404			{object}	gin.H
//	@Failure		500			{object}	gin.H
//	@Router			/user/findMask [post]
func FindUserByMask(c *gin.Context) {
	jsonData, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error ": err.Error()})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error ": err.Error()})
		return
	}
	var usr UserMask
	err = json.Unmarshal(jsonData, &usr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error ": err.Error()})
		return
	}
	type Result struct {
		User_name    string
		User_surname string
	}
	array, err := Finder(usr.Name, usr.Surname)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error ": err.Error()})
		return
	}
	var result []Result
	for _, val := range array {
		var res Result
		errs := db.Raw("SELECT user_name, user_surname FROM users WHERE id = ?", val).Scan(&res)
		if errs.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error ": errs.Error.Error()})
			return
		}
		if len(res.User_name) == 0 {
			errs := db.Raw("SELECT user_name, user_surname FROM users WHERE id = ?", val).Scan(&res)
			if errs.Error != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error ": errs.Error.Error()})
				return
			}
		}
		result = append(result, res)
	}
	c.JSON(200, gin.H{"found user(s) ": result})
}

// CreateReport godoc
//
//	@Summary		Create New Report
//	@Description	Create New Report
//	@Accept			json
//	@Produce		json
//
//	@Tags			mai lab API
//
//	@Param			user_log	body		Report	true	"Report's data"
//	@Success		200			{object}	string
//	@Failure		400			{object}	gin.H
//	@Failure		404			{object}	gin.H
//	@Failure		500			{object}	gin.H
//	@Router			/report/create [post]
func CreateReport(c *gin.Context) {
	jsonData, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error ": err.Error()})
		return
	}
	var rep Report
	err = json.Unmarshal(jsonData, &rep)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error ": err.Error()})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error ": err.Error()})
		return
	}

	err_exec := db.Exec("insert into reports (report_name, id) "+
		"VALUES (?, ?);", rep.Name, rep.UserId)
	if err_exec.Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error ": err.Error()})
		return
	}
	c.JSON(200, "report has been created")
}

// CreateConference godoc
//
//	@Summary		Create New Conference
//	@Description	Create New Conference
//	@Accept			json
//	@Produce		json
//
//	@Tags			mai lab API
//
//	@Param			conference_name	path		string	true	"conference name"
//	@Success		200				{object}	string
//	@Failure		400				{object}	gin.H
//	@Failure		404				{object}	gin.H
//	@Failure		500				{object}	gin.H
//	@Router			/conference/create/{conference_name} [post]
func CreateConference(c *gin.Context) {
	conf := c.Params.ByName("conference_name")
	if len(conf) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error ": "conference name not provided"})
		return
	}

	err_exec := db.Exec("insert into conferences (conference_name) "+
		"VALUES (?);", conf)
	if err_exec.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error ": err_exec.Error.Error()})
		return
	}
	c.JSON(200, "conference has been created")
}

// GetAllReports godoc
//
//	@Summary		Get All Reports
//	@Description	Get All Reports
//	@Accept			json
//	@Produce		json
//
//	@Tags			mai lab API
//
//	@Success		200	{object}	[]string
//	@Failure		400	{object}	gin.H
//	@Failure		404	{object}	gin.H
//	@Failure		500	{object}	gin.H
//	@Router			/report/getAll [get]
func GetAllReports(c *gin.Context) {

	var res []string
	rows, err := db.Table("reports").Select("reports.report_name").Rows()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error ": err.Error()})
		return
	}
	for rows.Next() {
		var tmp string
		err := rows.Scan(&tmp)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error ": err.Error()})
			return
		}
		res = append(res, tmp)
	}
	if len(res) == 0 {
		c.JSON(404, "there are zero reports")
		return
	}
	c.JSON(200, gin.H{"found reports(s) ": res})
}

// AddReport godoc
//
//	@Summary		Add New Report
//	@Description	Add New Report
//	@Accept			json
//	@Produce		json
//
//	@Tags			mai lab API
//
//	@Param			conference_id	path		string	true	"conference id"
//	@Param			report_id		path		string	true	"report id"
//	@Success		200				{object}	string
//	@Failure		400				{object}	gin.H
//	@Failure		404				{object}	gin.H
//	@Failure		500				{object}	gin.H
//	@Router			/conference/addReport/{conference_id}/{report_id}/ [post]
func AddReport(c *gin.Context) {

	rep_id := c.Params.ByName("report_id")
	cf_id := c.Params.ByName("conference_id")
	if len(rep_id) == 0 || len(cf_id) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error ": "id not provided"})
		return
	}
	err_exec := db.Exec("UPDATE reports "+
		"SET conference_id=? "+
		"WHERE report_id=?;", cf_id, rep_id)
	if err_exec.Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error ": err_exec.Error.Error()})
		return
	}
	c.JSON(200, "report has been added to conference")
}

// GetAllReportsInConfs godoc
//
//	@Summary		Get All Reports In Conference
//	@Description	Get All Reports In Conference
//	@Accept			json
//	@Produce		json
//	@Param			conference_id	path	string	true	"conference id"
//
//	@Tags			mai lab API
//
//	@Success		200	{object}	[]string
//	@Failure		400	{object}	gin.H
//	@Failure		404	{object}	gin.H
//	@Failure		500	{object}	gin.H
//	@Router			/conference/getAllReports/{conference_id}/ [get]
func GetAllReportsInConf(c *gin.Context) {

	cf_id := c.Params.ByName("conference_id")
	if len(cf_id) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error ": "id not provided"})
		return
	}
	rows, err := db.Table("reports").Select("reports.report_name").Where("conference_id = ?", cf_id).Rows()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error ": err.Error()})
		return
	}
	var res []string
	for rows.Next() {
		var tmp string
		err := rows.Scan(&tmp)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error ": err.Error()})
			return
		}
		res = append(res, tmp)
	}
	if len(res) == 0 {
		c.JSON(404, "no reports found in this conference")
		return
	}
	c.JSON(200, gin.H{"found reports(s) in conference": res})
}
