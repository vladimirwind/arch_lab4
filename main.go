package main

import (
	_ "arch_lab/docs"
	"arch_lab/internal/handlers"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

//	@title			Swagger Example API
//	@version		1.0
//	@description	This is a simple Conference service.
//	@termsOfService	http://swagger.io/terms/
//	@Tags			mai lab API
//	@contact.name	Vladimir Vetrov
//	@contact.url	http://www.swagger.io/support
//	@contact.email	support@swagger.io

//	@license.name	Apache 2.0
//	@license.url	http://www.apache.org/licenses/LICENSE-2.0.html

//	@host		localhost:8080
//	@BasePath	/api/v1

//	@securityDefinitions.basic	BasicAuth

// @externalDocs.description	OpenAPI
// @externalDocs.url			https://swagger.io/resources/open-api/
func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	env := os.Getenv("ENVIRONMENT")
	router := gin.Default()

	router.POST("/user/create", handlers.CreateUser)
	router.GET("/user/findLogin/:user_log", handlers.FindUserByLogin)
	router.POST("/user/findMask", handlers.FindUserByMask)
	router.POST("/report/create", handlers.CreateReport)
	router.GET("/report/getAll", handlers.GetAllReports)
	router.GET("/user/findById/:id", handlers.FindUserByID)
	router.POST("/conference/create/:conference_name", handlers.CreateConference)
	router.POST("/conference/addReport/:conference_id/:report_id/", handlers.AddReport)
	router.GET("/conference/getAllReports/:conference_id", handlers.GetAllReportsInConf)
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	time.Sleep(5 * time.Second)
	_, err = handlers.ConnectDB()
	if err != nil {
		log.Panicln(err.Error())
	}
	err = handlers.ConnectRedis()
	if err != nil {
		log.Panicln(err.Error())
	}
	if env == "test" {
		router.Run("localhost:8080")
	} else {
		router.Run("0.0.0.0:8080")
	}
}
