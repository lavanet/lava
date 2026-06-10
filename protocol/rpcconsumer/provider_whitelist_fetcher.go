package rpcconsumer

import (
	"context"
	"os"
	"time"

	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/protocol/lavasession"
	"github.com/lavanet/lava/v5/utils"
	"github.com/lavanet/lava/v5/utils/specfetcher"
)

// providerWhitelistRetryInterval is the short interval used to retry the initial whitelist load
// until it first succeeds. This avoids a long fail-open window: if the source is briefly
// unreachable at startup, we keep retrying quickly instead of waiting a full refresh interval.
const providerWhitelistRetryInterval = 30 * time.Second

// ProviderWhitelistFetcher periodically loads the consumer provider whitelist from its source and
// applies it to the shared *lavasession.ProviderWhitelist. The source is either:
//   - a GitHub/GitLab directory URL, fetched with the exact same machinery as specs
//     (specfetcher); or
//   - a local JSON file path.
//
// For remote sources it authenticates with the dedicated --providers-whitelist-token when set,
// otherwise it falls back to the shared --github-token / --gitlab-token (picked by provider),
// matching how specs are authenticated. This lets the whitelist live in a different repo with a
// different credential than the specs.
//
// A single fetcher serves all chains (the whitelist is global; each per-chain session manager
// queries it with its own ChainID). It is only constructed/started when a source is configured,
// so when the flag is empty no refresh loop runs at all.
type ProviderWhitelistFetcher struct {
	source          string
	whitelistToken  string // dedicated token for the whitelist source; falls back to github/gitlab token when empty
	githubToken     string
	gitlabToken     string
	refreshInterval time.Duration
	target          *lavasession.ProviderWhitelist
}

func NewProviderWhitelistFetcher(source, whitelistToken, githubToken, gitlabToken string, refreshInterval time.Duration, target *lavasession.ProviderWhitelist) *ProviderWhitelistFetcher {
	return &ProviderWhitelistFetcher{
		source:          source,
		whitelistToken:  whitelistToken,
		githubToken:     githubToken,
		gitlabToken:     gitlabToken,
		refreshInterval: refreshInterval,
		target:          target,
	}
}

// Start performs an initial load (retrying on a short interval until the first success) and then
// refreshes on refreshInterval until ctx is cancelled. Intended to run in its own goroutine.
func (f *ProviderWhitelistFetcher) Start(ctx context.Context) {
	if f.source == "" || f.target == nil {
		return
	}
	// Defensive floor: guarantees a valid ticker interval even if this fetcher is constructed
	// directly with a zero/negative interval (the cobra wiring already applies the same default).
	if f.refreshInterval <= 0 {
		f.refreshInterval = common.DefaultProvidersWhitelistRefreshInterval
	}

	utils.LavaFormatInfo("starting provider whitelist fetcher",
		utils.LogAttr("source", f.source),
		utils.LogAttr("refresh_interval", f.refreshInterval),
	)

	// Initial load: retry quickly until the first success so the consumer doesn't relay in
	// passthrough for a whole refresh interval if the source is briefly unavailable at startup.
	if !f.loadOnce(ctx) && !f.retryInitialLoad(ctx) {
		// ctx was cancelled before the first successful load.
		return
	}

	ticker := time.NewTicker(f.refreshInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			f.loadOnce(ctx)
		}
	}
}

// retryInitialLoad polls the source on a short interval until the first load succeeds, returning
// true once it does. It returns false if ctx is cancelled first. The retry ticker is scoped to this
// function so it stops as soon as the initial load completes, rather than ticking uselessly for the
// rest of the fetcher's lifetime.
func (f *ProviderWhitelistFetcher) retryInitialLoad(ctx context.Context) bool {
	retryInterval := providerWhitelistRetryInterval
	if f.refreshInterval > 0 && f.refreshInterval < retryInterval {
		retryInterval = f.refreshInterval
	}
	retry := time.NewTicker(retryInterval)
	defer retry.Stop()
	for {
		select {
		case <-ctx.Done():
			return false
		case <-retry.C:
			if f.loadOnce(ctx) {
				return true
			}
		}
	}
}

// loadOnce fetches the whitelist from its source and applies it to the target, returning true on
// success. On failure the previous (last-known-good) whitelist is left intact; the consumer keeps
// using it, and if nothing has loaded yet it stays in passthrough.
func (f *ProviderWhitelistFetcher) loadOnce(ctx context.Context) bool {
	if specfetcher.IsRemoteRepoURL(f.source) {
		return f.loadFromRemote(ctx)
	}
	return f.loadFromFile()
}

func (f *ProviderWhitelistFetcher) loadFromRemote(ctx context.Context) bool {
	files, err := specfetcher.FetchAllFilesFromRemote(ctx, f.source, f.resolveTokenForSource())
	if err != nil {
		f.logLoadFailure("failed fetching provider whitelist from remote", err)
		return false
	}
	if err := f.target.UpdateFromFiles(files); err != nil {
		f.logLoadFailure("failed parsing provider whitelist from remote", err)
		return false
	}
	return true
}

// logLoadFailure logs a load/refresh failure with wording that reflects what the consumer actually
// does next, which differs by lifecycle stage:
//   - before the first successful load (startup) nothing is in place, so the consumer stays in
//     passthrough and relays to ALL providers until a load succeeds;
//   - after a list has loaded, the failure is non-destructive: the last-known-good snapshot is kept
//     and keeps gating relays.
func (f *ProviderWhitelistFetcher) logLoadFailure(msg string, err error) {
	if f.target != nil && f.target.Enabled() {
		utils.LavaFormatError(msg+"; keeping the last-known-good whitelist", err, utils.LogAttr("source", f.source))
		return
	}
	utils.LavaFormatError(msg+"; no whitelist loaded yet, consumer is ALLOWING ALL relays (passthrough) until the first successful load", err, utils.LogAttr("source", f.source))
}

// resolveTokenForSource returns the auth token to use for the whitelist remote fetch. The
// dedicated --providers-whitelist-token wins when set; otherwise it falls back to the shared
// --github-token / --gitlab-token, picked by the source's provider (mirroring how specs are
// authenticated in statetracker.loadAllSpecsFromRemoteRepo).
func (f *ProviderWhitelistFetcher) resolveTokenForSource() string {
	if f.whitelistToken != "" {
		return f.whitelistToken
	}
	if specfetcher.IsGitHubURL(f.source) {
		return f.githubToken
	}
	if specfetcher.IsGitLabURL(f.source) {
		return f.gitlabToken
	}
	return ""
}

func (f *ProviderWhitelistFetcher) loadFromFile() bool {
	content, err := os.ReadFile(f.source)
	if err != nil {
		f.logLoadFailure("failed reading provider whitelist file", err)
		return false
	}
	if err := f.target.UpdateFromBytes(content); err != nil {
		f.logLoadFailure("failed parsing provider whitelist file", err)
		return false
	}
	return true
}
