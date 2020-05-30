package main

import (
	"fmt"
	"os"
	"release"

	flag "github.com/spf13/pflag"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	incrementFormat = "%03d"
)

func main() {

	var module, remote, message string
	var verbose, dryRun, doPush bool
	format := "%Y.%m."
	flag.StringVarP(&module, "component", "c", "", "component to tag, if not set will use 'release' which triggers all components to release")
	flag.StringVarP(&remote, "remote", "r", "origin", "git remote to push to (if --push)")
	flag.StringVarP(&message, "msg", "m", "", "optional release message, will create an annotated git tag")
	// flag.StringVarP(&format, "fmt", "f", "%Y.%m.", "date format to use")
	flag.BoolVarP(&verbose, "verbose", "v", false, "enable more output")
	flag.BoolVar(&doPush, "push", false, "push tag to default remote (does 'git push')")
	flag.BoolVarP(&dryRun, "dry-run", "n", false, "don't create a release, just print what would be released")
	flag.Parse()

	if module == "" {
		module = "release"
	}

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	// If we want UTC use this
	// zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if verbose {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	cwd, err := os.Getwd()
	release.CheckIfError(err, "failed to get current dir")

	// Create a new Release Manager
	rm, err := release.NewManager(cwd, format, incrementFormat)

	// This is customizable, but for now, we always want a release number
	rm.AlwaysIncludeNumber = true

	release.CheckIfError(err, "failed to load release manager")
	next, _ := rm.GetProposedName(module)
	if dryRun {
		fmt.Printf("would create release:\n%s\n", next)
		os.Exit(0)
	}
	_, err = rm.CreateTag(next, message)
	if err != nil {
		log.Fatal().Msgf("failed to create tag %s: %s", next, err.Error())
		os.Exit(1)
	}

	fmt.Printf("created release: %s\n", next)
	if doPush {
		msg, err := rm.PushTagToRemote(next, remote)
		if err == nil {
			// Great Success!
			fmt.Println(msg)
		} else {
			log.Error().Err(err).Msg(msg)
		}
	} else if verbose {
		fmt.Printf("\nto push, just run:\n")
		fmt.Printf("git push %s %s\n", remote, next)
		fmt.Printf("OR\n")
		fmt.Printf("git push %s --tags\n", remote)
	}

}
