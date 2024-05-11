package main

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
)

type PersonInfo struct {
	Name        string `json:"name"`
	PhoneNumber string `json:"phone_number"`
	City        string `json:"city"`
	State       string `json:"state"`
	Street1     string `json:"street1"`
	Street2     string `json:"street2"`
	ZipCode     string `json:"zip_code"`
}

type PersonRequest struct {
	Name        string `json:"name"`
	PhoneNumber string `json:"phone_number"`
	City        string `json:"city"`
	State       string `json:"state"`
	Street1     string `json:"street1"`
	Street2     string `json:"street2"`
	ZipCode     string `json:"zip_code"`
}

func main() {
	db, err := sql.Open("mysql", "root:123@tcp(127.0.0.1:3306)/cetec")
	if err != nil {
		panic(err.Error())
	}
	defer db.Close()

	router := gin.Default()

	router.GET("/person/:person_id/info", func(c *gin.Context) {
		personID, err := strconv.Atoi(c.Param("person_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid person ID"})
			return
		}

		var personInfo PersonInfo

		// Query to fetch person info, phone number, and address details
		err = db.QueryRow("SELECT p.name, ph.number, a.city, a.state, a.street1, a.street2, a.zip_code "+
			"FROM person p "+
			"JOIN phone ph ON p.id = ph.person_id "+
			"JOIN address_join aj ON p.id = aj.person_id "+
			"JOIN address a ON aj.address_id = a.id "+
			"WHERE p.id = ?", personID).
			Scan(&personInfo.Name, &personInfo.PhoneNumber, &personInfo.City, &personInfo.State, &personInfo.Street1, &personInfo.Street2, &personInfo.ZipCode)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch person info"})
			return
		}

		c.JSON(http.StatusOK, personInfo)
	})

	router.POST("/person/create", func(c *gin.Context) {
		var personReq PersonRequest
		if err := c.BindJSON(&personReq); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		// Start a transaction
		tx, err := db.Begin()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
			return
		}
		defer func() {
			if r := recover(); r != nil {
				tx.Rollback()
			}
		}()

		// Insert into address table
		res, err := tx.Exec("INSERT INTO address (city, state, street1, street2, zip_code) VALUES (?, ?, ?, ?, ?)",
			personReq.City, personReq.State, personReq.Street1, personReq.Street2, personReq.ZipCode)
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert into address table"})
			return
		}
		addressID, err := res.LastInsertId()
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get last inserted ID"})
			return
		}

		// Insert into person table
		res, err = tx.Exec("INSERT INTO person (name) VALUES (?)", personReq.Name)
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert into person table"})
			return
		}
		personID, err := res.LastInsertId()
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get last inserted ID"})
			return
		}

		// Insert into phone table
		_, err = tx.Exec("INSERT INTO phone (person_id, number) VALUES (?, ?)", personID, personReq.PhoneNumber)
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert into phone table"})
			return
		}

		// Insert into address_join table
		_, err = tx.Exec("INSERT INTO address_join (person_id, address_id) VALUES (?, ?)", personID, addressID)
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert into address_join table"})
			return
		}

		// Commit transaction
		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
			return
		}

		c.Status(http.StatusOK)
	})

	router.Run(":8080")
}
