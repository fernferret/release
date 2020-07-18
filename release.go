package release

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
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

var pat = regexp.MustCompile(`^(?P<year>\d{4})\.(?P<month>\d{2})\.(?P<release>\d{3,})-.*$`)

type calVerStandard struct {
	Year    uint64
	Month   uint64
	Release uint64
}

func newCalVerStandard(year, month, rel uint64) *calVerStandard {
	return &calVerStandard{
		Year:    year,
		Month:   month,
		Release: rel,
	}
}

func (c *calVerStandard) String() string {
	return fmt.Sprintf("Release: %d.%02d.%03d", c.Year, c.Month, c.Release)
}

func (c *calVerStandard) FormatRelease(release string) string {
	return fmt.Sprintf("%d.%02d.%03d-%s", c.Year, c.Month, c.Release, release)
}

func (c *calVerStandard) IsAfter(other *calVerStandard) bool {
	// Check to see of the other is greater than us, return the opposite of that
	return !(other.Year > c.Year || other.Month > c.Month || other.Release > c.Release)
}

func (c *calVerStandard) IsSameMonth(other *calVerStandard) bool {
	return other.Year == c.Year && other.Month == c.Month
}

func (c *calVerStandard) Increase() *calVerStandard {
	c.Release++
	return c
}

func (r *Manager) getNextDateString(name string, now time.Time) string {
	// Create a new calVerStandard object to use as a baseline comparison. We do
	// this with a 0 release time so this function can blindly call .Increase()
	// at the end and not have to deal with a case where we created our own
	// versus a case where we found another tag. If we find one (say .023) we'll
	// have to increase it, but I want to reduce the branches so I just set this
	// to 0, so the default entry will be 001
	latest := newCalVerStandard(uint64(now.Year()), uint64(now.Month()), 0)
	for _, release := range r.releases {
		if pat.MatchString(release.Tag) {
			results := pat.FindStringSubmatch(release.Tag)
			year, _ := strconv.ParseUint(results[1], 10, 64)
			month, _ := strconv.ParseUint(results[2], 10, 64)
			relNum, _ := strconv.ParseUint(results[3], 10, 64)
			rev := newCalVerStandard(year, month, relNum)
			// Make sure the tag we're comparing is of our YYYY.MM, if it's not,
			// we don't even bother comparing, we're not interested in past or
			// future releases.
			if !rev.IsSameMonth(latest) {
				// Future time
				continue
			}
			if rev.IsAfter(latest) {
				latest = rev
			}
		}
	}

	// Always increase the release before returning, this way we always get a
	// unique one.
	return latest.Increase().FormatRelease(name)
}

// GetProposedName returns a proposed name for the next release tag
func (r *Manager) GetProposedName(name string) string {
	now := time.Now()
	return r.getNextDateString(name, now)
}
