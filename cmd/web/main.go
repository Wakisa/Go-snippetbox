package main

import (
	"database/sql"
	"flag"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"time"

	// Import the models package that we just created. You need to prefix
	// this with what module path you set up back(Project setup and Creating
	// a Module) so that the import statement looks like this:
	// "{your-module-path}/internal/models". If you can't remember that module
	// path you used, you can find it at the top of the go.mod file.
	"wakisa.com/internal/models"

	"github.com/alexedwards/scs/mysqlstore"
	"github.com/alexedwards/scs/v2"
	"github.com/go-playground/form/v4"
	_ "github.com/go-sql-driver/mysql"
)

// Define an application struct to hold the application dependecies
// for the web applicaion. For now we'll only include the structured
// logger, but we'll add more to this as the build progresses.
// Add a templateCache field to the application struct.
type application struct {
	logger         *slog.Logger
	snippets       *models.SnippetModel
	templateCache  map[string]*template.Template
	formDecoder    *form.Decoder
	sessionManager *scs.SessionManager
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

	// Use the scs.New() function to initalize a new session manager. Then we
	// configure it to use our MYSQL database as the session store, and set a
	// lifetime of 12 hours (so that sessions automatically expire 12 hours
	// after first being created).
	sessionManager := scs.New()
	sessionManager.Store = mysqlstore.New(db)
	sessionManager.Lifetime = 12 * time.Hour
	//Make sure that the Secure attributeis set on our session cookies.
	// Setting this means that the cookie will only be sent by a user's web
	// browser when a HTTPS connection is being used (and won't be sent over an
	// unsecure HTTP connection).
	sessionManager.Cookie.Secure = true

	// And add the session manager to our application dependencies.
	app := &application{
		logger:         logger,
		snippets:       &models.SnippetModel{DB: db},
		templateCache:  templateCache,
		formDecoder:    formDecoder,
		sessionManager: sessionManager,
	}

	// Initialize a new http.Server struct. We set the Addr and Handler fields so
	// that the server uses the same network address and routes as before.
	srv := &http.Server{
		Addr:    *addr,
		Handler: app.routes(),
		// Create a *log.Logger from our structured logger handler, which writes
		// log entries at Error level, and assign it to the ErrorLog field. If
		// you would prefer to log the server error at Warn level instead, you
		// could pass slog.LevelWarn as the final parameter.
		ErrorLog: slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	logger.Info("starting server", "addr", srv.Addr)

	// Use the ListenAndServeTLS() method to start the HTTPS sever. We
	// pass in the paths to the TLS certificate and corrensponding private key as
	// the two parameters.
	err = srv.ListenAndServeTLS("./tls/cert.pem", "./tls/key.pem")
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
