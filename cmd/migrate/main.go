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
	"github.com/spf13/cobra"
)

func main() {
	log.SetPrefix("migrate: ")
	log.SetFlags(0)

	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "migrate",
		Short: "Apply MongoDB schema migrations",
		Long: `Applies MongoDB schema migrations via golang-migrate (JSON runCommand migrations).

Requires MONGO_URI (admin-privileged URI; database segment should be jobby).
Optional MIGRATIONS_PATH (defaults to ./migrations).`,
	}

	root.AddCommand(
		newUpCmd(),
		newDownCmd(),
		newVersionCmd(),
		newForceCmd(),
	)

	return root
}

func newUpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "up",
		Short: "Apply all pending migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withMigrate(func(mig *migrate.Migrate) error {
				if err := mig.Up(); err != nil {
					if errors.Is(err, migrate.ErrNoChange) {
						log.Println("no change (already up to date)")
						return nil
					}
					return fmt.Errorf("up: %w", err)
				}
				return nil
			})
		},
	}
}

func newDownCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "down",
		Short: "Roll back one migration step",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withMigrate(func(mig *migrate.Migrate) error {
				if err := mig.Steps(-1); err != nil {
					if errors.Is(err, migrate.ErrNoChange) {
						log.Println("no change (nothing to roll back)")
						return nil
					}
					return fmt.Errorf("down: %w", err)
				}
				return nil
			})
		},
	}
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the current migration version",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withMigrate(func(mig *migrate.Migrate) error {
				v, dirty, err := mig.Version()
				if err != nil {
					if errors.Is(err, migrate.ErrNilVersion) {
						log.Println("version: none (fresh database)")
						return nil
					}
					return fmt.Errorf("version: %w", err)
				}
				state := ""
				if dirty {
					state = " (dirty)"
				}
				log.Printf("version: %d%s", v, state)
				return nil
			})
		},
	}
}

func newForceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "force <version>",
		Short: "Set migration version manually (recovery only)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			v, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("force: invalid version: %w", err)
			}
			return withMigrate(func(mig *migrate.Migrate) error {
				if err := mig.Force(v); err != nil {
					return fmt.Errorf("force: %w", err)
				}
				return nil
			})
		},
	}
}

func withMigrate(fn func(mig *migrate.Migrate) error) error {
	cfg, err := loadMigrateConfig()
	if err != nil {
		return err
	}

	absRoot, err := filepath.Abs(filepath.Clean(cfg.MigrationsPath))
	if err != nil {
		return err
	}

	mig, err := migrate.New(migrationsSourceURL(absRoot), cfg.URI)
	if err != nil {
		return fmt.Errorf("init migrations: %w", err)
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

	return fn(mig)
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
