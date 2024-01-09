package main

import (
	"github.com/gin-gonic/gin"
	"log"
)

func main() {
	router := gin.New()
	jobHandler := NewJobHandler()

	router.POST("/jobs", jobHandler.ParseStream)
	router.GET("/jobs/:id", jobHandler.Status)
	router.GET("/jobs/:id/result", jobHandler.Result)
	router.DELETE("/jobs/:id", jobHandler.Delete)

	err := router.Run(":80")
	if err != nil {
		log.Fatal(err)
	}
}
