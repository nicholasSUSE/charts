package main

import (
	"context"
	"flag"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	repov1 "helm.sh/helm/v3/pkg/repo"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/yaml"
)

var (
	cwd = func() string {
		d, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
		return d
	}()
	indexPath  = flag.String("helm-repo-index", filepath.Join(cwd, "index.yaml"), "path to index.yaml")
	entriesDir = flag.String("entries-dir", filepath.Join(cwd, "entries"), "destination directory")
)

func main() {
	flag.Parse()

	ctx := signals.SetupSignalHandler()
	if err := run(ctx); err != nil {
		log.Fatal(err)
	}
}

type ReleaseEntryAlias struct {
	*repov1.ChartVersion
	Omit bool `json:"omit"`
}

func run(ctx context.Context) error {
	index := repov1.NewIndexFile()

	rootDir := os.DirFS(*entriesDir)
	entries, err := fs.Glob(rootDir, "*/*.yaml")
	if err != nil {
		return err
	}
	for _, p := range entries {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		data, err := fs.ReadFile(rootDir, p)
		if err != nil {
			return err
		}

		var entry ReleaseEntryAlias
		if err := yaml.Unmarshal(data, &entry); err != nil {
			return err
		}
		if entry.Omit {
			continue
		}

		name := filepath.Base(filepath.Dir(p))
		index.Entries[name] = append(index.Entries[name], entry.ChartVersion)
	}
	index.SortEntries()

	return index.WriteFile(*indexPath, 0644)
}
