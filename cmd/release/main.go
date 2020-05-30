package main

import (
	"fmt"
	"os"
	"release"

	"github.com/go-git/go-git/v5/config"
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
	var user, email string
	format := "%Y.%m."
	flag.StringVarP(&module, "component", "c", "", "component to tag, if not set will use 'release' which triggers all components to release")
	flag.StringVarP(&remote, "remote", "r", "origin", "git remote to push to (if --push)")
	flag.StringVarP(&message, "msg", "m", "", "optional release message, will create an annotated git tag")
	flag.StringVar(&user, "user", "", "override user in ~/.gitconfig")
	flag.StringVar(&email, "email", "", "override email in ~/.gitconfig")
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

	cfg, err := config.LoadConfig(config.GlobalScope)
	if err == nil {
		if user == "" {
			user = cfg.User.Name
		}
		if email == "" {
			email = cfg.User.Email
		}
	} else {
		// At this point, we might be in a CI environment and might not have gitconfig
		// setup. If we're not using heavy tags, we don't even care about this error,
		// so we'll log a warning (only visible at debug) and if the user tries to create
		// an annotated tag, we'll deal with it then.
		log.Debug().Err(err).Msg("unable to load git config, this is only a problem if you're using annotated tags")
	}

	cwd, err := os.Getwd()
	release.CheckIfError(err, "failed to get current dir")

	// Create a new Release Manager
	rm, err := release.NewManager(cwd, format, incrementFormat)

	// This is customizable, but for now, we always want a release number
	rm.AlwaysIncludeNumber = true

	release.CheckIfError(err, "failed to load release manager")
	newRelease, _ := rm.GetProposedName(module)
	if dryRun {
		fmt.Printf("would create release:\n%s\n", newRelease)
		os.Exit(0)
	}
	_, err = rm.CreateTag(newRelease, message, user, email)
	if err != nil {
		log.Fatal().Msgf("failed to create tag %s: %s", newRelease, err.Error())
		os.Exit(1)
	}

	fmt.Printf("created release: %s\n", newRelease)
	if doPush {
		msg, err := rm.PushTagToRemote(newRelease, remote)
		if err == nil {
			// Great Success!
			fmt.Println(msg)
		} else {
			log.Error().Err(err).Msg(msg)
		}
	} else if verbose {
		fmt.Printf("\nto push, just run:\n")
		fmt.Printf("git push %s %s\n", remote, newRelease)
		fmt.Printf("OR\n")
		fmt.Printf("git push %s --tags\n", remote)
	}

}
