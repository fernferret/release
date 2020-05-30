package release

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/cactus/gostrftime"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/rs/zerolog/log"
)

// CheckIfError checks if the given error is nil, if not it prints a message and exits
func CheckIfError(err error, msg string) {
	if err == nil {
		return
	}

	log.Fatal().Err(err).Msg(msg)
}

// Release represents a release
type Release struct {
	Tag            string            // The human readable name of the tag
	Hash           string            // The hash of the git commit that the tag points to
	ReleaseMessage string            // This is the tag message, might not be present
	CommitMessage  string            // The commit message should always be present
	Author         object.Signature  // The author of the tag
	Committer      object.Signature  // The committer (person who merged/ran git commit)
	Tagger         *object.Signature // The person who created a proper tag (will be nil for lightweight tags)
}

// Date returns the date of when the commit the tag points to happened
func (r *Release) Date() time.Time {
	return r.Committer.When
}

// ReleasedBy returns who the tag was released by.
// It looks at the Tagger first, if nil, it defaults to the Committer
func (r *Release) ReleasedBy() object.Signature {
	if r.Tagger != nil {
		return *r.Tagger
	}
	return r.Committer
}

// ReleasedByString gives a nice printable string of who performed the release
func (r *Release) ReleasedByString(includeDate bool) string {
	relBy := r.ReleasedBy()
	if includeDate {
		return fmt.Sprintf("%s <%s> on %s", relBy.Name, relBy.Email, gostrftime.Format("%Y-%m-%d %H:%M:%S", relBy.When))
	}
	return fmt.Sprintf("%s <%s>", relBy.Name, relBy.Email)
}

// Message returns a friendly messaage for the commit, it uses the tagged message if that's
// available and defaults to the commit message
func (r *Release) Message() string {
	if r.ReleaseMessage != "" {
		return r.ReleaseMessage
	}
	return strings.SplitN(r.CommitMessage, "\n", 1)[0]
}

type releaseList []Release

func (s releaseList) Len() int {
	return len(s)
}
func (s releaseList) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s releaseList) Less(i, j int) bool {
	if s[i].Date().Equal(s[j].Date()) {
		return strings.Compare(s[i].Tag, s[j].Tag) == -1
	}
	return s[i].Date().After(s[j].Date())
}

// Manager is responsible for keeping the state required to perform releases
type Manager struct {
	// Git Items
	repoDir             string
	cwd                 string
	repo                *git.Repository
	releases            releaseList
	timeFmt             string
	incFmt              string
	AlwaysIncludeNumber bool
}

// FindRepoDir finds a git repository directory in the current or any parent directory
func FindRepoDir(path string) (string, error) {
	path = filepath.Clean(path)
	if path == "/" {
		return path, fmt.Errorf("no git directory found in any parent directory")
	}
	gitPath := filepath.Join(path, ".git")
	if _, err := os.Stat(gitPath); os.IsNotExist(err) {
		return FindRepoDir(filepath.Dir(path))
	}
	return path, nil
}

// NewManager creates a new release manager with a given directory
func NewManager(cwd, timeFmt, incFmt string) (*Manager, error) {
	repoDir, err := FindRepoDir(cwd)
	log.Debug().Msgf("searching for git directory in: %s", cwd)
	CheckIfError(err, "failed to find repo dir")
	r, err := git.PlainOpen(repoDir)
	CheckIfError(err, "failed to load git repository")

	mgr := &Manager{
		repoDir: repoDir,
		cwd:     cwd,
		repo:    r,
		timeFmt: timeFmt,
		incFmt:  incFmt,
	}
	mgr.loadGitTags()
	return mgr, nil
}

func tagToRefspec(tag string) config.RefSpec {
	return config.RefSpec(fmt.Sprintf("refs/tags/%s:refs/tags/%s", tag, tag))
}

// CheckRemote performs a basic existence check on the remote and returns an error if there is a problem
func (r *Manager) CheckRemote(remote string) error {
	_, err := r.repo.Remote(remote)
	return err
}

// PushTagToRemote pushes the given local tag to the remote repository
// returns a message to be displayed to the user along with an an optional error,
// If err is nil, the operation was successful
func (r *Manager) PushTagToRemote(tag, remote string, auth transport.AuthMethod) (string, error) {
	options := &git.PushOptions{
		RemoteName: remote,
		RefSpecs: []config.RefSpec{
			tagToRefspec(tag),
		},
		Auth: auth,
	}
	err := r.repo.Push(options)
	if err == git.NoErrAlreadyUpToDate {
		return fmt.Sprintf("nothing pushed, tag %s already existed and was up to date in remote %s", tag, remote), nil
	} else if err != nil {
		return fmt.Sprintf("failed to push tag %s to remote %s", tag, remote), err
	}
	return fmt.Sprintf("pushed tag %s to remote %s", tag, remote), err
}

func (r *Manager) loadGitTags() {
	tagrefs, err := r.repo.Tags()
	CheckIfError(err, "failed to load lightweight tags")
	// Reset the relesae list
	r.releases = releaseList{}
	tagrefs.ForEach(func(t *plumbing.Reference) error {
		newRelease := Release{}
		obj, err := r.repo.CommitObject(t.Hash())
		if err != nil {
			tag, _ := r.repo.TagObject(t.Hash())
			newRelease.Tag = tag.Name
			newRelease.ReleaseMessage = tag.Message
			newRelease.Tagger = &tag.Tagger
			obj, err = tag.Commit()
			if err != nil {
				log.Error().Err(err).Msgf("failed to load commit for tag %s, this looks bad, skipping", tag.Name)
				return nil
			}
		} else {
			newRelease.Tag = t.Name().String()[10:]
		}
		newRelease.Hash = obj.ID().String()
		newRelease.CommitMessage = obj.Message
		newRelease.Author = obj.Author
		newRelease.Committer = obj.Committer
		r.releases = append(r.releases, newRelease)
		log.Debug().Str("hash", newRelease.Hash).Str("releaser", newRelease.ReleasedByString(true)).Msgf("loaded tag: %s", newRelease.Tag)
		return nil
	})
	sort.Sort(r.releases)
}

// CreateTag creates a tag in the repo, if comment is specified it creates an annotated tag
func (r *Manager) CreateTag(name, comment, user, email string) (*plumbing.Reference, error) {
	hash, err := r.repo.Head()
	if err != nil {
		return nil, err
	}
	var opts *git.CreateTagOptions
	if comment != "" {
		if user == "" || email == "" {
			msg := "both user and email are required when specifying a message, something might be wrong with your ~/.gitconfig or you didn't specify --name and --email"
			log.Fatal().Str(
				"name", user,
			).Str(
				"email", email,
			).Msg(msg)
		}
		sig := &object.Signature{
			Name:  user,
			Email: email,
			When:  time.Now(),
		}
		opts = &git.CreateTagOptions{Message: comment, Tagger: sig}
	}
	return r.repo.CreateTag(name, hash.Hash(), opts)
}

// GetProposedName returns a proposed name for the next release tag
func (r *Manager) GetProposedName(name string) (string, []string) {
	tried := []string{}
	now := time.Now()
	proposedDate := gostrftime.Strftime(r.timeFmt, now)
	pfx := ""
	proposedName := fmt.Sprintf("%s%s", pfx, proposedDate)
	// Iterate thorugh tags finding the latest cantidate release name
	idx := 0
	// Sometimes, we'll want to always include a release, this will give us:
	// 2020.01.001, 2020.01.002 instead of 2020.01, 2020.01.001
	if r.AlwaysIncludeNumber {
		idx = 1
	}
	tried = append(tried, proposedName)
	for {
		// The first time we won't do anything, so you'll never get 20.05.0
		// Instead you'll get the following:
		// 2020.05
		// 2020.05.1
		// 2020.05.2
		// ...
		// Sometimes it may be preferable to force an increment
		if idx > 0 {
			// If we have an index
			proposedFmt := fmt.Sprintf("%%s%s", r.incFmt)
			proposedName = fmt.Sprintf(proposedFmt, proposedDate, idx)
			tried = append(tried, proposedName)
		}
		foundTag := false
		for _, release := range r.releases {
			if strings.Contains(release.Tag, proposedName) {
				// Already have a release
				foundTag = true
				break
			}
		}
		if foundTag {
			idx++
			continue
		}
		if name != "" && !strings.HasPrefix(name, "-") {
			name = fmt.Sprintf("-%s", name)
		}
		return fmt.Sprintf("%s%s", proposedName, name), tried
	}
}
