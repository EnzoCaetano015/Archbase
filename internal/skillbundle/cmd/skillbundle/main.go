// skillbundle is a development utility. It is intentionally separate from arc.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/EnzoCaetano015/Archbase/internal/skillbundle"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: skillbundle lock|validate|package|eval-suite|eval-fixture|eval-prepare|eval-check [flags]")
	}
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	var options skillbundle.Options
	flags.StringVar(&options.Root, "root", "", "skill root (explicit)")
	flags.StringVar(&options.Manifest, "manifest", "", "skill manifest JSON (explicit)")
	flags.StringVar(&options.Lock, "lock", "", "catalog lock JSON (explicit)")
	flags.StringVar(&options.CatalogRoot, "catalog-root", "", "local OSS Git checkout (explicit)")
	output := flags.String("output", "", "new output file or directory; never overwritten")
	commit := flags.String("commit", "", "full catalog commit")
	tag := flags.String("tag", "", "catalog tag")
	suite := flags.String("suite", "", "evaluation suite JSON")
	results := flags.String("results", "", "evaluation result directory")
	scenario := flags.String("scenario", "", "scenario ID")
	skillVersion := flags.String("skill-version", "", "skill version")
	skillHash := flags.String("skill-sha256", "", "skill ZIP SHA-256")
	release := flags.Bool("release", false, "require every scenario to have passed")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments")
	}
	require := func(values ...string) error {
		for _, value := range values {
			if value == "" {
				return fmt.Errorf("%s: required flags missing; see -h", args[0])
			}
		}
		return nil
	}
	switch args[0] {
	case "lock":
		if err := require(options.CatalogRoot, *tag, *commit, *output); err != nil {
			return err
		}
		lock, err := skillbundle.CreateLock(context.Background(), options.CatalogRoot, *tag, *commit)
		if err != nil {
			return err
		}
		return skillbundle.WriteJSON(*output, lock)
	case "validate", "package":
		if err := require(options.Root, options.Manifest, options.Lock, options.CatalogRoot); err != nil {
			return err
		}
		bundle, err := skillbundle.Prepare(context.Background(), options)
		if err != nil {
			return err
		}
		if args[0] == "package" {
			if err = require(*output); err != nil {
				return err
			}
			return bundle.Write(*output)
		}
		fmt.Printf("valid: %s %s (%d files)\n", bundle.Manifest.Name, bundle.Manifest.Version, len(bundle.Files))
		return nil
	case "eval-suite":
		if err := require(*suite); err != nil {
			return err
		}
		loaded, err := skillbundle.LoadSuite(*suite)
		if err == nil {
			fmt.Printf("valid suite: %d scenarios\n", len(loaded.Scenarios))
		}
		return err
	case "eval-fixture":
		if err := require(*suite, *scenario, *output); err != nil {
			return err
		}
		return skillbundle.MaterializeFixture(*suite, *scenario, *output)
	case "eval-prepare":
		if err := require(*suite, *scenario, *skillVersion, *skillHash, *commit, *output); err != nil {
			return err
		}
		return skillbundle.PrepareEvaluation(*suite, *scenario, *skillVersion, *skillHash, *commit, *output)
	case "eval-check":
		if err := require(*suite, *results); err != nil {
			return err
		}
		if *release {
			return skillbundle.CheckEvaluations(*suite, *results, true, skillbundle.EvaluationIdentity{SkillVersion: *skillVersion, SkillSHA256: *skillHash, CatalogCommit: *commit})
		}
		return skillbundle.CheckEvaluations(*suite, *results, false)
	default:
		return fmt.Errorf("unknown operation %s", args[0])
	}
}
