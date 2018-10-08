package main

import (
	"io/ioutil"
	"net/http"
	"os"
	"time"

	log "github.com/cihub/seelog"
	"github.com/gin-gonic/gin"
)

type PostSearch struct {
	SearchTerm string `json:"search"`
	DataURL    string `json:"url"`
}

func main() {
	port := "8185"
	setLogLevel("debug")

	// setup gin server
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	// Standardize logs
	r.Use(middleWareHandler(), gin.Recovery())

	// Example for binding JSON ({"user": "manu", "password": "123"})
	r.POST("/search", func(c *gin.Context) {
		err, pages := func(c *gin.Context) (err error, pages []db.Page) {
			var ps PostSearch
			err = c.ShouldBindJSON(&ps)
			if err != nil {
				return nil, err
			}
			// get data URL
			tmpfile, err := ioutil.TempFile("", "example")
			if err != nil {
				log.Error(err)
			}
			defer os.Remove(tmpfile.Name()) // clean up

			return
		}(c)

		if err != nil {
			message := err.Error()
		}
		c.JSON(http.StatusOK, gin.H{"success": err == nil, "message": err.Error(), "pages": pages})
	})
	log.Infof("Running at http://0.0.0.0:" + port)
	err := r.Run(":" + port)
	if err != nil {
		log.Error(err)
	}
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
