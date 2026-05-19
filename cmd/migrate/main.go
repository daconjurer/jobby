// Command migrate applies MongoDB schema migrations via golang-migrate/migrate (JSON runCommand migrations).
//
// Requires MONGO_URI (admin-privileged URI, target database segment should be jobby — see migrations/README.md).
// Optional MIGRATIONS_PATH (defaults to ./migrations).
//
// Commands: up, down, version, force <version>
package main

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mongodb"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	log.SetPrefix("migrate: ")
	log.SetFlags(0)

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: migrate <up|down|version|force> [args]")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "environment:")
		fmt.Fprintln(os.Stderr, "  MONGO_URI           required MongoDB URI (dbname path should be jobby)")
		fmt.Fprintln(os.Stderr, "  MIGRATIONS_PATH     migrations directory (default ./migrations)")
		os.Exit(2)
	}

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		log.Fatal("MONGO_URI environment variable is required")
	}

	migDir := os.Getenv("MIGRATIONS_PATH")
	if migDir == "" {
		migDir = "./migrations"
	}
	absRoot, err := filepath.Abs(filepath.Clean(migDir))
	if err != nil {
		log.Fatal(err)
	}

	srcURL := migrationsSourceURL(absRoot)

	mig, err := migrate.New(srcURL, mongoURI)
	if err != nil {
		log.Fatalf("init migrations: %v", err)
	}
	defer func() {
		srcErr, dbErr := mig.Close()
		if srcErr != nil {
			log.Printf("close source driver: %v", srcErr)
		}
		if dbErr != nil {
			log.Printf("close database driver: %v", dbErr)
		}
	}()

	switch os.Args[1] {
	case "up":
		if err := mig.Up(); err != nil {
			if errors.Is(err, migrate.ErrNoChange) {
				log.Println("no change (already up to date)")
				return
			}
			log.Fatalf("up: %v", err)
		}
	case "down":
		if err := mig.Steps(-1); err != nil {
			if errors.Is(err, migrate.ErrNoChange) {
				log.Println("no change (nothing to roll back)")
				return
			}
			log.Fatalf("down: %v", err)
		}
	case "version":
		v, dirty, err := mig.Version()
		if err != nil {
			if errors.Is(err, migrate.ErrNilVersion) {
				log.Println("version: none (fresh database)")
				return
			}
			log.Fatalf("version: %v", err)
		}
		state := ""
		if dirty {
			state = " (dirty)"
		}
		log.Printf("version: %d%s", v, state)
	case "force":
		if len(os.Args) != 3 {
			log.Fatal("usage: migrate force <version>")
		}
		v, err := strconv.Atoi(os.Args[2])
		if err != nil {
			log.Fatalf("force: invalid version: %v", err)
		}
		if err := mig.Force(v); err != nil {
			log.Fatalf("force: %v", err)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", os.Args[1])
		os.Exit(2)
	}
}

// migrationsSourceURL builds a file:// URL for golang-migrate's file source driver.
func migrationsSourceURL(absDir string) string {
	p := filepath.ToSlash(absDir)
	u := url.URL{
		Scheme: "file",
		Path:   p,
	}
	return u.String()
}
