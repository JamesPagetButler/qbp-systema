// Package arxiv implements the QBP arXiv daily-batch scout's Fetch +
// Transform halves per QBP federation tenancy v0.2 §4.1.
//
// Sub-issue closure (Fetch half): QBP #448 (W5.1).
//
// # Contract
//
// Fetch implements the wyrd/scoutd.Scout.Fetch interface (Wyrd PR #54 §2):
//
//	type Scout struct {
//	    ...
//	    Fetch func(ctx context.Context) ([]byte, error)
//	    ...
//	}
//
// The function produces raw arXiv Atom XML response bytes from the 13
// declared sources per tenancy §4.1; the daemon then dispatches the
// returned bytes to Transform (transform.go, W5.2) which mints NT_SIGNAL
// hyperedges.
//
// # Federation-tenancy contract requirements (from tenancy v0.2 §4.1)
//
//   - Rate-limit: 3 requests/second (arXiv API policy)
//   - User-Agent: "QBP-Federation-Tenant/0.2 (https://github.com/JamesPagetButler/QBP)"
//   - Backoff: exponential with jitter; max 5 retries; ceiling 30s
//   - Sources: 13 arXiv categories per tenancy §4.1 list
//   - Respects ctx cancellation
//
// # Status
//
// SKELETON STAGED IN /tmp/qbp-sprint2-staging/ — committed to qbp-systema
// at Sprint 2 kickoff (post-Beekeeper close-out sequence) per Phase 2 of
// /home/prime/.claude/plans/misty-chasing-clover.md. Won't compile until:
//
//   - wyrd/scoutd impl PR merges (Wyrd PR #54 → impl PR; tracked QBP #443)
//   - qbp-systema repo exists with go.mod referencing github.com/JamesPagetButler/wyrd at the Compute Manifest current SHA
//
// # F4 / F6 from my Wyrd PR #54 review
//
// Wyrd PR #54 F6 proposed a helper `scoutd/util.RateLimitedFetch(rps, ua,
// backoffPolicy)`. If Wyrd ships the helper at v0.1, this Fetch wraps raw
// arXiv HTTP calls with it. If not, the rate-limit + UA + backoff are
// implemented inline below (current skeleton uses inline implementation).
package arxiv

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// userAgent is the arXiv API User-Agent per federation tenancy v0.2 §4.1.
const userAgent = "QBP-Federation-Tenant/0.2 (https://github.com/JamesPagetButler/QBP)"

// arxivAPIBase is the arXiv API endpoint root.
const arxivAPIBase = "https://export.arxiv.org/api/query"

// sources is the 13 arXiv categories declared in tenancy v0.2 §4.1.
// Loaded by the scout daemon at startup; passed to Fetch on each cycle.
// Exported so tests can override.
var sources = []string{
	"astro-ph.HE",       // high-energy astrophysics
	"astro-ph.CO",       // cosmology
	"astro-ph.SR",       // solar/stellar
	"quant-ph",          // quantum physics
	"hep-th",            // high-energy theory
	"hep-ph",            // high-energy phenomenology
	"cond-mat.mes-hall", // mesoscopic + nano
	"cond-mat.supr-con", // superconductivity
	"cond-mat.str-el",   // strongly correlated
	"math.QA",           // quantum algebra
	"math.MP",           // mathematical physics
	"math.OA",           // operator algebra
	"gr-qc",             // general relativity + quantum cosmology
}

// rateLimit is the arXiv API rate-limit per tenancy §4.1.
const rateLimit = 3 // requests per second

// rateLimitInterval is computed from rateLimit; minimum inter-request delay.
var rateLimitInterval = time.Second / time.Duration(rateLimit)

// BackoffPolicy bundles the exponential-backoff-with-jitter parameters.
type BackoffPolicy struct {
	MaxRetries int
	BaseDelay  time.Duration
	Ceiling    time.Duration
}

// DefaultBackoff matches tenancy §4.1: max 5 retries; ceiling 30s.
var DefaultBackoff = BackoffPolicy{
	MaxRetries: 5,
	BaseDelay:  100 * time.Millisecond,
	Ceiling:    30 * time.Second,
}

// httpClient is the package-level HTTP client; exported for tests to swap.
var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

// AtomFeed is the minimal arXiv response shape Fetch returns and Transform
// consumes. Atom 1.0 per https://www.w3.org/2005/Atom; arXiv-specific
// extensions are in the arxiv: namespace.
type AtomFeed struct {
	XMLName xml.Name    `xml:"feed"`
	Entries []AtomEntry `xml:"entry"`
}

// AtomEntry is one arXiv paper.
type AtomEntry struct {
	ID         string         `xml:"id"`
	Title      string         `xml:"title"`
	Summary    string         `xml:"summary"`
	Published  string         `xml:"published"`
	Updated    string         `xml:"updated"`
	Authors    []AtomAuthor   `xml:"author"`
	Categories []AtomCategory `xml:"category"`
}

// AtomAuthor is one paper author.
type AtomAuthor struct {
	Name string `xml:"name"`
}

// AtomCategory is one arXiv subject classification.
type AtomCategory struct {
	Term string `xml:"term,attr"`
}

// Fetch implements wyrd/scoutd.Scout.Fetch for the QBP arXiv daily-batch
// scout. Polls all 13 source categories sequentially under the rate-limit;
// concatenates the Atom XML responses into a single byte stream (one
// `<feed>` per source); returns to the daemon for Transform dispatch.
//
// Returns ErrFetchAllSourcesFailed if every source poll fails (network
// outage, arXiv API down). Returns partial success silently if some
// sources succeed and others fail — partial data flows to Transform with
// per-entry source attribution; daemon's TotalCycles metric counts the
// cycle once regardless of per-source failures.
func Fetch(ctx context.Context) ([]byte, error) {
	var (
		results       [][]byte
		failedSources []string
	)

	for i, source := range sources {
		// Per-request rate-limiting (sequential; no concurrent requests
		// to arXiv; conservative against API policy).
		if i > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(rateLimitInterval):
			}
		}

		body, err := fetchSourceWithBackoff(ctx, source, DefaultBackoff)
		if err != nil {
			failedSources = append(failedSources, fmt.Sprintf("%s: %s", source, err))
			continue
		}
		results = append(results, body)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("%w: all %d sources failed: %s",
			ErrFetchAllSourcesFailed, len(sources), strings.Join(failedSources, "; "))
	}

	// Concatenate Atom feeds with explicit source-separator comments so
	// Transform can attribute entries to source categories. The Atom
	// parser ignores XML comments; the separator is informational.
	var combined strings.Builder
	for i, body := range results {
		combined.Write(body)
		combined.WriteString(fmt.Sprintf("\n<!-- end-of-source-%d -->\n", i))
	}
	return []byte(combined.String()), nil
}

// fetchSourceWithBackoff issues one arXiv API query for a single source
// with exponential backoff + jitter on transient failures.
func fetchSourceWithBackoff(ctx context.Context, source string, b BackoffPolicy) ([]byte, error) {
	q := url.Values{}
	q.Set("search_query", "cat:"+source)
	q.Set("sortBy", "submittedDate")
	q.Set("sortOrder", "descending")
	q.Set("max_results", "200") // per-source cap; daemon aggregates across sources

	requestURL := arxivAPIBase + "?" + q.Encode()

	var lastErr error
	for attempt := 0; attempt <= b.MaxRetries; attempt++ {
		if attempt > 0 {
			// Backoff: BaseDelay * 2^(attempt-1) + jitter; capped at Ceiling.
			delay := b.BaseDelay * (1 << (attempt - 1))
			if delay > b.Ceiling {
				delay = b.Ceiling
			}
			// Jitter: ±25% uniform.
			jitter := time.Duration(rand.Int63n(int64(delay) / 2)) //nolint:gosec // jitter not cryptographic
			delay = delay - delay/4 + jitter
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
		if err != nil {
			return nil, fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("User-Agent", userAgent)
		req.Header.Set("Accept", "application/atom+xml")

		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}
		// Treat 2xx as success; retry 429 + 5xx; fail-fast on 4xx other than 429.
		switch {
		case resp.StatusCode >= 200 && resp.StatusCode < 300:
			return body, nil
		case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500:
			lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
		default:
			return nil, fmt.Errorf("HTTP %d (non-retryable): %s", resp.StatusCode, string(body))
		}
	}
	return nil, fmt.Errorf("after %d retries: %w", b.MaxRetries, lastErr)
}

// ErrFetchAllSourcesFailed is returned when every per-source poll failed.
// Wraps the underlying per-source error list as context.
var ErrFetchAllSourcesFailed = errors.New("arxiv fetch: all sources failed")
