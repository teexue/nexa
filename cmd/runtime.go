package main

import (
	"fmt"
	"log/slog"

	"github.com/teexue/common-agent/core/config"
	"github.com/teexue/common-agent/core/i18n"
	"github.com/teexue/nexakit/provider"
	"github.com/teexue/common-agent/core/store"
)

type runtimePaths struct {
	home      string
	agentsDir string
}

func defaultPaths() (runtimePaths, error) {
	home, err := config.Home(false)
	if err != nil {
		return runtimePaths{}, err
	}
	return runtimePaths{home: home, agentsDir: config.AgentsDir(home)}, nil
}

func resolvePaths(homeFlag string) (runtimePaths, error) {
	if homeFlag != "" {
		return runtimePaths{home: homeFlag, agentsDir: config.AgentsDir(homeFlag)}, nil
	}
	return defaultPaths()
}

func openStateDB(home string, logger *slog.Logger) (*store.DB, error) {
	if err := config.EnsureDirs(home); err != nil {
		return nil, err
	}
	db, err := store.Open(home)
	if err != nil {
		return nil, err
	}
	config.BindDB(db)
	logger.Debug("log.runtime.state_db", "path", store.StateFile(home))
	return db, nil
}

func bootstrapRuntime(paths runtimePaths, useMock bool, logger *slog.Logger) (*provider.Catalog, *config.CredentialStore, *store.DB, error) {
	if useMock {
		return nil, nil, nil, nil
	}
	db, err := openStateDB(paths.home, logger)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("open state.db: %w", err)
	}

	creds, err := config.NewCredentialStore(paths.home)
	if err != nil {
		_ = db.Close()
		return nil, nil, nil, fmt.Errorf("load credentials: %w (run: common-agent config init)", err)
	}

	catalog, err := db.LoadCatalog(creds.Lookup)
	if err != nil {
		if provider.IsMissingCatalogError(err) {
			logger.Warn("log.runtime.no_providers", "path", "state.db")
			return nil, creds, db, nil
		}
		_ = db.Close()
		return nil, nil, nil, fmt.Errorf("load providers: %w (run: common-agent config init)", err)
	}
	logger.Debug("log.runtime.ready", "home", paths.home)
	return catalog, creds, db, nil
}

func printPaths(paths runtimePaths) {
	fmt.Println(i18n.T("cli.paths.home", "path", paths.home))
	fmt.Println(i18n.T("cli.paths.agents", "path", paths.agentsDir))
	fmt.Println(i18n.T("cli.paths.state", "path", store.StateFile(paths.home)))
}
