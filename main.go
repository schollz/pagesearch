package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"time"

	log "github.com/cihub/seelog"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/schollz/pagesearch/db"
	"github.com/schollz/pagesearch/dl"
)

type PostSearch struct {
	SearchTerm string `json:"search"`
	DataURL    string `json:"url"`
}

func main() {
	port := "8185"
	setLogLevel("debug")

	os.Mkdir("data", 0644)
	files, err := ioutil.ReadDir("data")
	if err != nil {
		log.Error(err)
		return
	}

	for _, f := range files {
		fmt.Println(f.Name(), f.ModTime())
	}

	// setup gin server
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	// Standardize logs
	r.Use(middleWareHandler(), gin.Recovery())

	// Example for binding JSON ({"user": "manu", "password": "123"})
	r.POST("/search", func(c *gin.Context) {
		searchTime := time.Now()
		pages, err := func(c *gin.Context) (pages []db.Page, err error) {
			var ps PostSearch
			err = c.ShouldBindJSON(&ps)
			if err != nil {
				return
			}
			dbName := path.Join("data", base64.URLEncoding.EncodeToString([]byte(ps.DataURL))+".pagename.db")
			var fs *db.Database
			if _, err = os.Stat(dbName); os.IsNotExist(err) {
				// db does not exist
				// get data URL
				tmpfile, err := ioutil.TempFile("", "pagesearch")
				if err != nil {
					log.Error(err)
				}
				defer os.Remove(tmpfile.Name())                            // clean up
				err = dl.DownloadFile(tmpfile.Name(), ps.DataURL, 1000000) // download file
				if err != nil {
					err = errors.Wrap(err, fmt.Sprintf("file must be less than %d bytes", 1000000))
					return pages, err
				}

				// attempt to open
				bJSON, err := ioutil.ReadFile(tmpfile.Name())
				if err != nil {
					err = errors.Wrap(err, "could not read tempfile")
					return pages, err
				}
				// check if it can be deciphered
				err = json.Unmarshal(bJSON, &pages)
				if err != nil {
					err = fmt.Errorf("incorrect format for pages")
					return pages, err
				}

				// open database
				log.Debugf("opening %s for %s", base64.URLEncoding.EncodeToString([]byte(ps.DataURL)), ps.DataURL)
				fs, err = db.New(dbName)
				if err != nil {
					return pages, err
				}

				// save the pages
				err = fs.SaveMany(pages)
				if err != nil {
					return pages, err
				}
			} else {
				log.Debugf("opening %s for %s", base64.URLEncoding.EncodeToString([]byte(ps.DataURL)), ps.DataURL)
				fs, err = db.New(dbName)
				if err != nil {
					return pages, err
				}
			}

			// search
			pages, err = fs.Find(ps.SearchTerm)
			if err != nil {
				pages = []db.Page{}
			}
			return
		}(c)

		message := fmt.Sprintf("found %d pages in %s", len(pages), time.Since(searchTime))
		if err != nil {
			message = err.Error()
		}
		c.JSON(http.StatusOK, gin.H{"success": err == nil, "message": message, "pages": pages})
	})
	log.Infof("Running at http://0.0.0.0:" + port)
	r.Run(":" + port)
}

func middleWareHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		t := time.Now()
		// Add base headers
		addCORS(c)
		// Run next function
		c.Next()
		// Log request
		log.Infof("%v %v %v %s", c.Request.RemoteAddr, c.Request.Method, c.Request.URL, time.Since(t))
	}
}

func addCORS(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	c.Writer.Header().Set("Access-Control-Max-Age", "86400")
	c.Writer.Header().Set("Access-Control-Allow-Methods", "GET")
	c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Max")
	c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
}

// setLogLevel determines the log level
func setLogLevel(level string) (err error) {

	// https://en.wikipedia.org/wiki/ANSI_escape_code#3/4_bit
	// https://github.com/cihub/seelog/wiki/Log-levels
	appConfig := `
	<seelog minlevel="` + level + `">
	<outputs formatid="stdout">
	<filter levels="debug,trace">
		<console formatid="debug"/>
	</filter>
	<filter levels="info">
		<console formatid="info"/>
	</filter>
	<filter levels="critical,error">
		<console formatid="error"/>
	</filter>
	<filter levels="warn">
		<console formatid="warn"/>
	</filter>
	</outputs>
	<formats>
		<format id="stdout"   format="%Date %Time [%LEVEL] %File %FuncShort:%Line %Msg %n" />
		<format id="debug"   format="%Date %Time %EscM(37)[%LEVEL]%EscM(0) %File %FuncShort:%Line %Msg %n" />
		<format id="info"    format="%Date %Time %EscM(36)[%LEVEL]%EscM(0) %File %FuncShort:%Line %Msg %n" />
		<format id="warn"    format="%Date %Time %EscM(33)[%LEVEL]%EscM(0) %File %FuncShort:%Line %Msg %n" />
		<format id="error"   format="%Date %Time %EscM(31)[%LEVEL]%EscM(0) %File %FuncShort:%Line %Msg %n" />
	</formats>
	</seelog>
	`
	logger, err := log.LoggerFromConfigAsBytes([]byte(appConfig))
	if err != nil {
		return
	}
	log.ReplaceLogger(logger)
	return
}
