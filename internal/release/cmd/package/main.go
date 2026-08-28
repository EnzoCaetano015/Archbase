package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	archrelease "github.com/EnzoCaetano015/Archbase/internal/release"
)

func main() {
	var tag, moduleRoot, outputDirectory, goBinary string
	var sourceEpoch int64
	flag.StringVar(&tag, "tag", "", "stable release tag, such as v0.1.0")
	flag.StringVar(&moduleRoot, "module-root", ".", "Archbase module root")
	flag.StringVar(&outputDirectory, "output", "dist", "new directory receiving release assets")
	flag.StringVar(&goBinary, "go", "go", "Go executable")
	flag.Int64Var(&sourceEpoch, "source-epoch", 0, "release commit timestamp as Unix seconds")
	flag.Parse()
	if sourceEpoch <= 0 {
		fail("-source-epoch must be a positive Unix timestamp")
	}
	assets, err := archrelease.Build(context.Background(), archrelease.Options{
		Tag: tag, ModuleRoot: moduleRoot, OutputDir: outputDirectory, GoBinary: goBinary,
		Timestamp: time.Unix(sourceEpoch, 0).UTC(),
	})
	if err != nil {
		fail(err.Error())
	}
	for _, asset := range assets {
		fmt.Println(asset)
	}
}

func fail(message string) {
	fmt.Fprintln(os.Stderr, "release package:", message)
	os.Exit(1)
}
