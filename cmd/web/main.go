package main

import (
	"database/sql"
	"flag"
	"html/template"
	"log/slog"
	"net/http"
	"os"

	// Import the models package that we just created. You need to prefix
	// this with what module path you set up back(Project setup and Creating
	// a Module) so that the import statement looks like this:
	// "{your-module-path}/internal/models". If you can't remember that module
	// path you used, you can find it at the top of the go.mod file.
	"wakisa.com/internal/models"

	"github.com/go-playground/form/v4"
	_ "github.com/go-sql-driver/mysql"
)

// Define an application struct to hold the application dependecies
// for the web applicaion. For now we'll only include the structured
// logger, but we'll add more to this as the build progresses.
// Add a templateCache field to the application struct.
type application struct {
	logger        *slog.Logger
	snippets      *models.SnippetModel
	templateCache map[string]*template.Template
	formDecoder   *form.Decoder
}

func main() {

	// Define a new command-line flag with the name 'addr', a
	// default value of ":4000" and some short help text explaining
	// what the flag controls. The value of the flag will be stored
	// in the addr variable at runtime.
	addr := flag.String("addr", ":4000", "HTTP network address")

	// Define a new command-line flag for the MYSQL DSN string
	dsn := flag.String("dsn", "web:pass@/snippetbox?parseTime=true", "MYSQL data source name")

	// Importantly, we use the flag.Parse() function to parse the command-line
	//flag. This reads in the command-line flag value and assigns it
	// to the addr variable. You need to call this *before* you use
	// the addr variable otherwise it will always contain the default
	// value of ":4000". If any errors are encoutentered during
	// parsing the application will be terminated.
	flag.Parse()

	// Use the slog.New() function to initialize a new structured logger
	// which writes to the standard out stream and uses the default settings
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// To keep the main() function tidy I've put the code for creating a
	// connection pool into the separate openDB() function below. We pass
	// openDB() the DSN from the command-line flag.
	db, err := openDB(*dsn)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	// We also defer a call to db.Close(), so that the connection pool is closed
	// before the main() function exits.
	defer db.Close()

	// Initialize a new template cache...
	templateCache, err := newTemplateCache()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	// Initializea decoder instance...
	formDecoder := form.NewDecoder()

	// And add it to the application dependencies.
	app := &application{
		logger:        logger,
		snippets:      &models.SnippetModel{DB: db},
		templateCache: templateCache,
		formDecoder:   formDecoder,
	}

	logger.Info("starting server", "addr", *addr)

	err = http.ListenAndServe(*addr, app.routes())
	logger.Error(err.Error())
	os.Exit(1)

}

// The openDB() function wraps sql.OPen() and returns a sql.DB connection pool
// for a given DSN.
func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}
