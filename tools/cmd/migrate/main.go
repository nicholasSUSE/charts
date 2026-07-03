package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"maps"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/sync/errgroup"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/registry"
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
	workingDir  = flag.String("workdir", "", "working directory")
	indexPath   = flag.String("helm-repo-index", filepath.Join(cwd, "index.yaml"), "path to index.yaml")
	ociRegistry = flag.String("registry", "oci://localhost:5000", "OCI registry address")
	parallel    = flag.Int("parallel", 1, "number of concurrent goroutines")
	entriesDir  = flag.String("entries-dir", filepath.Join(cwd, "entries"), "destination directory")
	skipPush    = flag.Bool("skip-push", false, "skip pushing changes to chart repo")
)

func main() {
	flag.Parse()
	if *parallel < 1 {
		*parallel = 1
	}

	ctx := signals.SetupSignalHandler()
	if err := run(ctx); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	index, err := repov1.LoadIndexFile(*indexPath)
	if err != nil {
		return fmt.Errorf("loading index file: %w", err)
	}

	if !*skipPush {
		baseDir := filepath.Dir(*indexPath)
		if *workingDir != "" {
			baseDir = *workingDir
		}
		if err := pushCharts(ctx, index, baseDir, *ociRegistry, *parallel); err != nil {
			return fmt.Errorf("pushing charts: %w", err)
		}
	}

	return createEntries(index, *entriesDir, *ociRegistry)
}

func pushCharts(ctx context.Context, index *repov1.IndexFile, baseDir string, ociRegistry string, parallel int) error {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}
	httpClient := &http.Client{Transport: transport}

	registryClient, err := registry.NewClient(
		registry.ClientOptHTTPClient(httpClient),
		registry.ClientOptDebug(true),
	)
	if err != nil {
		return fmt.Errorf("creating registry client: %w", err)
	}

	actionConfig := new(action.Configuration)
	actionConfig.RegistryClient = registryClient

	pushAction := action.NewPushWithOpts(
		action.WithPushConfig(actionConfig),
		action.WithInsecureSkipTLSVerify(true),
	)

	fmt.Printf("Found %d chart entries in index.\n", len(index.Entries))

	eg, ctx := errgroup.WithContext(ctx)
	eg.SetLimit(parallel)
	for _, chartName := range slices.Sorted(maps.Keys(index.Entries)) {
		eg.Go(func() error {
			if ctx.Err() != nil {
				return nil
			}
			fmt.Printf("Processing chart: %s\n", chartName)
			for _, chartVersion := range index.Entries[chartName] {
				if len(chartVersion.URLs) == 0 {
					log.Printf("Warning: No URLs found for %s-%s, skipping", chartName, chartVersion.Version)
					continue
				}

				// Assume the first URL is the relative local path to the .tgz package
				chart := filepath.Join(baseDir, chartVersion.URLs[0])

				// Verify the file actually exists before attempting to push
				if _, err := os.Open(chart); os.IsNotExist(err) {
					log.Printf("Error: Chart package not found at %q, skipping", chart)
					continue
				} else if err != nil {
					return err
				}

				fmt.Printf("  -> Pushing version %s (%s) to the registry...\n", chartVersion.Version, filepath.Base(chart))

				if output, err := pushAction.Run(chart, ociRegistry); err != nil {
					return fmt.Errorf("pushing chart: %w\n%v", err, output)
				}
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}

	fmt.Println("Migration process complete!")
	return nil

}

func createEntries(index *repov1.IndexFile, baseDir string, ociRegistry string) error {
	for chartName, versions := range index.Entries {
		chartURI, err := url.JoinPath(ociRegistry, chartName)
		if err != nil {
			return err
		}
		chartDir := filepath.Join(baseDir, chartName)
		if err := os.MkdirAll(chartDir, 0755); err != nil {
			return err
		}
		for _, version := range versions {
			tag := strings.ReplaceAll(version.Version, "+", "_")
			version.URLs = []string{chartURI + ":" + tag}
			data, err := yaml.Marshal(version)
			if err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(chartDir, version.Version+".yaml"), data, 0644); err != nil {
				return err
			}
		}
	}
	return nil
}
