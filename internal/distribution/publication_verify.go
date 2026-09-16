package distribution

import (
	"context"
	"errors"
	"io"
	"reflect"
	"regexp"
	"strconv"
	"time"
)

type PublicationSelection struct {
	Version      string `json:"version"`
	Tag          string `json:"tag"`
	Identity     string `json:"identity"`
	SourceCommit string `json:"source_commit"`
}

var candidateIdentityPattern = regexp.MustCompile(`^mandalore:([0-9a-f]{40}):sha256:[0-9a-f]{64}$`)

// SelectPublication validates explicit coordinates, not their provenance,
// acceptance or existence. It performs no network access or other effects.
func SelectPublication(version, identity string) (PublicationSelection, error) {
	match := candidateIdentityPattern.FindStringSubmatch(identity)
	if !ValidVersion(version) || len(match) != 2 {
		return PublicationSelection{}, errors.New("publication requires a canonical version and exact candidate identity")
	}
	return PublicationSelection{Version: version, Tag: "v" + version, Identity: identity, SourceCommit: match[1]}, nil
}

// VerifyPublication is anonymous and read-only. It re-fetches the actual release,
// source tag and every asset; a saved publication receipt is not verification.
// It never configures policy, executes payloads, publishes or closes issues.
func (c *ReleaseClient) VerifyPublication(ctx context.Context, version, identity string) (receipt PublicationReceipt, err error) {
	if _, err := SelectPublication(version, identity); err != nil {
		return receipt, err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	verified, err := c.inspectVersion(ctx, version)
	if err != nil {
		return receipt, err
	}
	view := verified.view
	if view.Manifest.Identity() != identity {
		return receipt, errors.New("published release does not match the selected candidate identity")
	}
	receipt = PublicationReceipt{Identity: identity, SourceCommit: view.Manifest.Manifest.SourceCommit, Phase: "published", ReleaseID: view.ID, URL: view.URL, Assets: view.Assets}
	path := "/releases/" + strconv.FormatInt(view.ID, 10)
	var current publicationMetadata
	if err := c.getJSON(ctx, path, &current); err != nil {
		return receipt, err
	}
	want := map[string]ReleaseAsset{}
	for _, a := range view.Assets {
		want[a.Name] = a
	}
	assets, err := publicationAssets(current, view.Manifest, want, true)
	if err != nil {
		return receipt, err
	}
	if current.ID != view.ID || *current.Draft || !reflect.DeepEqual(assets, view.Assets) {
		return receipt, errors.New("publication changed during verification")
	}
	for _, a := range assets {
		// Inspection already downloaded and hashed these exact metadata assets.
		// Reuse only its original bytes after the publication inventory matches.
		var checked []byte
		switch a.Name {
		case "manifest.json":
			checked = verified.manifest
		case "SHA256SUMS":
			checked = verified.checksums
		}
		if checked != nil {
			if int64(len(checked)) != a.Size || Digest(checked) != a.SHA256 {
				return receipt, errors.New("verified metadata bytes disagree with publication")
			}
			continue
		}
		if err := c.asset(ctx, a, io.Discard); err != nil {
			return receipt, err
		}
	}
	var refreshed publicationMetadata
	if err := c.getJSON(ctx, path, &refreshed); err != nil {
		return receipt, err
	}
	if !reflect.DeepEqual(refreshed, current) {
		return receipt, errors.New("publication changed during asset downloads")
	}
	commit, err := c.commit(ctx, view.Manifest.Manifest.Tag)
	if err != nil {
		return receipt, err
	}
	if commit != receipt.SourceCommit {
		return receipt, errors.New("published source tag changed during asset downloads")
	}
	if err := ctx.Err(); err != nil {
		return receipt, err
	}
	receipt.Phase = "publication-verified"
	return receipt, nil
}
