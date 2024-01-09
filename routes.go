package main

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jkittell/data/database"
	"go.mongodb.org/mongo-driver/bson"
	"log"
	"net/http"
	"time"
)

type JobHandler struct {
	jobs    database.MongoDB[Job]
	results database.MongoDB[Result]
}

func NewJobHandler() *JobHandler {
	jobs, err := database.NewMongoDB[Job]("jobs")
	if err != nil {
		log.Fatal(err)
	}
	results, err := database.NewMongoDB[Result]("results")
	if err != nil {
		log.Fatal(err)
	}
	return &JobHandler{
		jobs:    jobs,
		results: results,
	}
}

func (h JobHandler) Run(job Job) {
	job.Status = "IN-PROGRESS"
	_, err := h.jobs.Update(context.TODO(), job.Id, job)
	if err != nil {
		log.Fatal(err)
		return
	}

	segments, err := GetSegments(job.URL)
	if err != nil {
		job.Error = err.Error()
	}

	res := Result{
		Id:       job.Id,
		URL:      job.URL,
		Segments: segments.ToSlice(),
	}

	err = h.results.Insert(context.TODO(), res)

	if err != nil {
		log.Fatal(err)
		return
	}
	job.Status = "COMPLETE"
	job.CompletedAt = time.Now()

	_, err = h.jobs.Update(context.TODO(), job.Id, job)
	if err != nil {
		log.Fatal(err)
		return
	}
}

func (h JobHandler) ParseStream(c *gin.Context) {
	data := struct {
		URL string `json:"url"`
	}{}

	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	job := Job{
		Id:          uuid.New(),
		URL:         data.URL,
		Status:      "SUBMITTED",
		CreatedAt:   time.Now(),
		CompletedAt: time.Time{},
	}

	err := h.jobs.Insert(c, job)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Location", fmt.Sprintf("/job/%s/status", job.Id))
	c.JSON(http.StatusAccepted, job)
	go h.Run(job)
}

func (h JobHandler) Status(c *gin.Context) {
	id := c.Param("id")
	val, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	job, err := h.jobs.FindByID(c, val)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, job)
}

func (h JobHandler) Result(c *gin.Context) {
	id := c.Param("id")
	val, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	res, err := h.results.FindByID(c, val)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h JobHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	val, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	err = h.results.Delete(c, bson.D{{Key: "id", Value: val}}, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
