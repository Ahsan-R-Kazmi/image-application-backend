package main

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"path/filepath"

	_ "github.com/lib/pq"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "postgres"
	dbname   = "image-application"
)

type FileInfo struct {
	Name string 	`json:"name"`
	IsFavorite bool `json:"isFavorite"`
	FilePath string `json:"filePath"`
}

const MaxMultipartFormMemory = 32 << 20
const StaticImageFileLocation = "http://localhost:8081/static/images"

// Create a global reference to the db connection, so that it can be used in other functions.
// https://stackoverflow.com/questions/40587008/how-do-i-handle-opening-closing-db-connection-in-a-go-app/40587071
var db *sql.DB

func main() {

	connectionString := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var err error
	db, err = sql.Open("postgres", connectionString)
	HandleError(err)

	// Defer closing the db connection until the main function exits.
	defer db.Close()

	router := gin.Default()
	router.Use(HandleCorsMiddleware)
	router.MaxMultipartMemory = MaxMultipartFormMemory

	router.Static("/static", "web/static/")
	fileApiV1 := router.Group("/api/v1/file")
	{
		fileApiV1.GET("/getAllFileInfo", HandleGetAllFileInfo)
		fileApiV1.POST("/upload", HandleFileUpload)
	}

	router.Run(":8081")
}

func HandleError(err error) {
	if err != nil {
		panic(err)
	}
}

// Allow all origins access, since the back-end application will not be accessible by the outside world.
// https://stackoverflow.com/questions/29418478/go-gin-framework-cors
func HandleCorsMiddleware(c *gin.Context)  {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
	c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, " +
		"Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
	c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")

	if c.Request.Method == "OPTIONS" {
		c.AbortWithStatus(204)
		return
	}

	c.Next()
}

func HandleGetAllFileInfo(c *gin.Context) {
	log.Println("HandleGetAllFileInfo attempting to get the information for all files.")

	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered in HandleGetAllFileInfo after trying to get all filenames. The following " +
				"error was encountered: ", r)

			c.String(http.StatusInternalServerError, fmt.Sprintf("Error in getting file info."))

			return
		}
	}()

	// Check that a connection to the database can be opened.
	err := db.Ping()
	HandleError(err)

	rows, err := db.Query("SELECT name, is_favorite FROM image_file")
	HandleError(err)

	defer rows.Close()

	var fileInfoList []FileInfo
	for rows.Next() {

		var fileName string
		var isFavorite bool
		err = rows.Scan(&fileName, &isFavorite)
		HandleError(err)

		fileInfo := FileInfo{
			fileName,
			isFavorite,
			StaticImageFileLocation + "/" + fileName,
		}

		fileInfoList = append(fileInfoList, fileInfo)
	}

	err = rows.Err()
	HandleError(err)

	log.Println("Returning response with file information.")
	c.JSON(http.StatusOK, fileInfoList)

	return
}


func HandleFileUpload(c *gin.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered in HandleFileUpload after trying to save uploaded file(s). The following " +
				"error was encountered: ", r)

			c.String(http.StatusInternalServerError, fmt.Sprintf("Error in uploading file(s)."))

			return
		}
	}()

	log.Println("Consuming uploaded files.")

	form, err := c.MultipartForm()
	if err != nil {
		c.String(http.StatusBadRequest,
			fmt.Sprintf("An error occurred while consuming the files:\n %s.", err.Error()))
	}
	files := form.File["files"]

	for index, file := range files {
		filename := filepath.Base(file.Filename)

		log.Printf("Consuming file %d of %d with name = %s to be downloaded to the database.",
			index, len(files), filename)

		err := SaveFileToDatabase(file, filename)
		HandleError(err)

		err = c.SaveUploadedFile(file, "web/static/images/" + filename)
		HandleError(err)
	}

	c.String(http.StatusOK, fmt.Sprintf("Succesfully uploaded file(s)."))
	return
}

func SaveFileToDatabase(file *multipart.FileHeader, filename string) error {

	// Check that a connection to the database can be opened.
	err := db.Ping()
	HandleError(err)

	// Check that no file with this name exists.
	row := db.QueryRow("SELECT COUNT(*) FROM image_file WHERE name = $1", filename)

	var count int
	err = row.Scan(&count)
	HandleError(err)

	if count >= 1 {
		return errors.New(fmt.Sprintf("File with name %s already exists.", filename))
	}

	// Populate the byte array with the file data.
	fileContent, err := file.Open()
	HandleError(err)

	byteArray, err := ioutil.ReadAll(fileContent)
	HandleError(err)

	insertStatement := "INSERT INTO image_file(name, data) VALUES($1, $2)"
	_, err = db.Exec(insertStatement, filename, byteArray)
	HandleError(err)

	log.Printf("Saved file with name = %s to the database.", filename)

	return nil
}