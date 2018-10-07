package db

import (
	"bufio"
	"compress/gzip"
	"database/sql"
	"encoding/json"
	"os"
	"sync"

	log "github.com/cihub/seelog"
	"github.com/pkg/errors"
	"github.com/schollz/sqlite3dump"
)

type Database struct {
	Name string
	DB   *sql.DB
	sync.RWMutex
}

// Page is the basic unit that is saved
type Page struct {
	ID       string            `json:"id"`
	MetaData map[string]string `json:"meta"`
	Data     string            `json:"data"`
}

// New will initialize a filesystem by creating DB and calling InitializeDB.
// Callers should ensure "github.com/mattn/go-sqlite3" is imported in some way
// before calling this so the sqlite3 driver is available.
func New(name string) (fs *Database, err error) {
	fs = new(Database)
	if name == "" {
		err = errors.New("database must have name")
		return
	}
	fs.Name = name

	fs.DB, err = sql.Open("sqlite3", fs.Name)
	if err != nil {
		return
	}
	err = fs.InitializeDB(true)
	if err != nil {
		err = errors.Wrap(err, "could not initialize")
		return
	}

	return
}

// InitializeDB will initialize schema if not already done and if dump is true,
// will create the an initial DB dump. This is automatically called by New.
func (fs *Database) InitializeDB(dump bool) (err error) {
	sqlStmt := `CREATE TABLE IF NOT EXISTS 
	fs (
		id TEXT NOT NULL PRIMARY KEY,
		meta TEXT
	);`
	_, err = fs.DB.Exec(sqlStmt)
	if err != nil {
		err = errors.Wrap(err, "creating table")
		return
	}

	sqlStmt = `CREATE VIRTUAL TABLE IF NOT EXISTS 
		fts USING fts4 (id,data);`
	_, err = fs.DB.Exec(sqlStmt)
	if err != nil {
		err = errors.Wrap(err, "creating virtual table")
	}
	return
}

// DumpSQL will dump the SQL as text to filename.sql.gz
func (fs *Database) DumpSQL() (err error) {
	fs.Lock()
	defer fs.Unlock()

	fi, err := os.Create(fs.Name + ".sql.gz")
	if err != nil {
		return
	}
	gf := gzip.NewWriter(fi)
	fw := bufio.NewWriter(gf)
	err = sqlite3dump.DumpDB(fs.DB, fw)
	fw.Flush()
	gf.Close()
	fi.Close()
	return
}

// NewPage returns a new file
func (fs *Database) NewPage(id, data string) (f Page) {
	f = Page{
		ID:   id,
		Data: data,
	}
	return
}

// Save a file to the file system. Will insert or ignore, and then update.
func (fs *Database) Save(f Page) (err error) {
	fs.Lock()
	defer fs.Unlock()

	tx, err := fs.DB.Begin()
	if err != nil {
		return errors.Wrap(err, "begin Save")
	}

	stmt, err := tx.Prepare(`
	INSERT OR IGNORE INTO
		fts
	(
		id,
		data
	) 
		values 	
	(
		?, 
		?
	)`)
	if err != nil {
		return errors.Wrap(err, "stmt Save")
	}

	_, err = stmt.Exec(
		f.ID,
		f.Data,
	)
	if err != nil {
		return errors.Wrap(err, "exec Save")
	}
	stmt.Close()

	stmt, err = tx.Prepare(`
	INSERT OR IGNORE INTO
		fs
	(
		id,
		meta
	) 
		values 	
	(
		?, 
		?
	)`)
	if err != nil {
		return errors.Wrap(err, "stmt Save")
	}
	bMeta, _ := json.Marshal(f.MetaData)
	_, err = stmt.Exec(
		f.ID,
		string(bMeta),
	)
	if err != nil {
		return errors.Wrap(err, "exec Save")
	}
	stmt.Close()

	err = tx.Commit()
	if err != nil {
		return errors.Wrap(err, "commit Save")
	}

	return
}

// Save many
func (fs *Database) SaveMany(pages []Page) (err error) {
	fs.Lock()
	defer fs.Unlock()

	tx, err := fs.DB.Begin()
	if err != nil {
		return errors.Wrap(err, "begin Save")
	}

	for _, f := range pages {
		err = func() error {
			stmt, err := tx.Prepare(`
	INSERT OR IGNORE INTO
		fts
	(
		id,
		data
	) 
		values 	
	(
		?, 
		?
	)`)
			defer stmt.Close()
			if err != nil {
				return errors.Wrap(err, "stmt Save")
			}

			_, err = stmt.Exec(
				f.ID,
				f.Data,
			)
			if err != nil {
				return errors.Wrap(err, "exec Save")
			}
			return nil
		}()
		if err != nil {
			return
		}
	}

	err = tx.Commit()
	if err != nil {
		return errors.Wrap(err, "commit Save")
	}

	return
}

// Close will make sure that the lock file is closed
func (fs *Database) Close() (err error) {
	return fs.DB.Close()
}

// Find returns the info from a file
func (fs *Database) Find(text string) (files []Page, err error) {
	fs.Lock()
	defer fs.Unlock()

	files, err = fs.getAllFromPreparedQuery(`
		SELECT id,snippet(fts,'<b>','</b>','...',-1,-30) 
			FROM fts
			WHERE data MATCH ?
	`, text)
	return
}

func (fs *Database) getAllFromPreparedQuery(query string, args ...interface{}) (files []Page, err error) {
	// prepare statement
	stmt, err := fs.DB.Prepare(query)
	if err != nil {
		err = errors.Wrap(err, "preparing query: "+query)
		return
	}

	defer stmt.Close()
	rows, err := stmt.Query(args...)
	if err != nil {
		err = errors.Wrap(err, query)
		return
	}

	// loop through rows
	defer rows.Close()
	files = []Page{}
	for rows.Next() {
		var f Page
		err = rows.Scan(
			&f.ID,
			&f.Data,
		)
		if err != nil {
			err = errors.Wrap(err, "get rows of file")
			return
		}
		files = append(files, f)
	}
	err = rows.Err()
	if err != nil {
		err = errors.Wrap(err, "getRows")
	}
	return
}

// setLogLevel determines the log level
func SetLogLevel(level string) (err error) {

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
		<format id="stdout"   format="%Date %Time [%LEVEL] %Page %FuncShort:%Line %Msg %n" />
		<format id="debug"   format="%Date %Time %EscM(37)[%LEVEL]%EscM(0) %Page %FuncShort:%Line %Msg %n" />
		<format id="info"    format="%Date %Time %EscM(36)[%LEVEL]%EscM(0) %Page %FuncShort:%Line %Msg %n" />
		<format id="warn"    format="%Date %Time %EscM(33)[%LEVEL]%EscM(0) %Page %FuncShort:%Line %Msg %n" />
		<format id="error"   format="%Date %Time %EscM(31)[%LEVEL]%EscM(0) %Page %FuncShort:%Line %Msg %n" />
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
