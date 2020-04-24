package stack

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/go-github/v29/github"
	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"
	sutils "github.com/kabanero-io/kabanero-operator/pkg/controller/stack/utils"
	cutils "github.com/kabanero-io/kabanero-operator/pkg/controller/utils"
	"sigs.k8s.io/controller-runtime/pkg/client"
	rlog "sigs.k8s.io/controller-runtime/pkg/log"
)

var gitCachelog = rlog.Log.WithName("gitcache")

// Value in the cache map.  This contains the etag returned from the remote
// server, which is used on subsequent requests to use the cached data.
type gitCacheData struct {
	assetId      int64
	size         int
	creationTime time.Time
	lastUsed     time.Time
	data         []byte
}

var gitCache = make(map[string]gitCacheData)

// The Duration at which a cache entry will be purged.
const gitPurgeDuration = 12 * time.Hour

// The amount of time between cache purge ticker cycles
const gitTickerDuration = 30 * time.Minute

// Mutex for concurrent map access
var gitCacheLock sync.Mutex

// Retrieves a stack index file content using GitHub APIs
func getStackDataUsingGit(c client.Client, gitRelease kabanerov1alpha2.GitReleaseInfo, skipCertVerification bool, namespace string, reqLogger logr.Logger) ([]byte, error) {

	// Get a Github client.
	gclient, err := getGitClient(c, gitRelease, skipCertVerification, namespace, reqLogger)
	if err != nil {
		return nil, err
	}

	// Get the release tagged in Github as repoConf.GitRelease.Release.
	release, response, err := gclient.Repositories.GetReleaseByTag(context.Background(), gitRelease.Organization, gitRelease.Project, gitRelease.Release)
	if err != nil || response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Unable to retrieve object representing Github repository release %v. Configured GitRelease data: %v. Error: %v", gitRelease.Release, gitRelease, err)
	}

	return getReleaseAsset(gclient, release.Assets, gitRelease)
}

// Retrieves a Git client.
func getGitClient(c client.Client, gitRelease kabanerov1alpha2.GitReleaseInfo, skipCertVerification bool, namespace string, reqLogger logr.Logger) (*github.Client, error) {
	var client *github.Client

	tlsConfig, err := cutils.GetTLSCConfig(c, skipCertVerification)
	if err != nil {
		return nil, err
	}

	transport := &http.Transport{TLSClientConfig: tlsConfig}

	// Search all secrets under the given namespace for the one containing the required hostname.
	annotationKey := "kabanero.io/git-"
	secret, err := cutils.GetMatchingSecret(c, namespace, sutils.SecretAnnotationFilter, gitRelease.Hostname, annotationKey)
	if err != nil {
		newError := fmt.Errorf("Unable to find secret matching annotation values: %v and %v in namespace %v Error: %v", annotationKey, gitRelease.Hostname, namespace, err)
		return nil, newError
	}

	var pat []byte
	if secret != nil {
		reqLogger.Info(fmt.Sprintf("Secret used for secured GIT client requests: %v. Secret annotations: %v", secret.GetName(), secret.Annotations))
		pat, _ = secret.Data["password"]
	}

	httpClient, err := cutils.GetHTTPClient(pat, transport)
	if err != nil {
		return nil, err
	}

	switch {
	// GHE.
	case gitRelease.Hostname != "github.com":
		// GHE hostnames must be suffixed with /api/v3/ otherwise 406 status codes
		// will be returned. Using NewEnterpriseClient will do that for us automatically.
		url := "https://" + gitRelease.Hostname
		eclient, err := github.NewEnterpriseClient(url, url, httpClient)
		if err != nil {
			return nil, err
		}
		client = eclient
	// Non GHE.
	default:
		client = github.NewClient(httpClient)
	}

	return client, nil
}

func getReleaseAsset(gclient *github.Client, assets []github.ReleaseAsset, gitRelease kabanerov1alpha2.GitReleaseInfo) ([]byte, error) {
	var indexBytes []byte

	// Find the asset identified as repoConf.GitRelease.AssetName and download it.
	for _, asset := range assets {
		if asset.GetName() == gitRelease.AssetName {
			path := fmt.Sprintf("%s:%s:%s:%s:%s", gitRelease.Hostname, gitRelease.Organization, gitRelease.Project, gitRelease.Release, gitRelease.AssetName)

			// Return the cached data if it was found in the cache and the current/cached asset IDs match.
			gitCacheLock.Lock()
			cacheData, found := gitCache[path]
			gitCacheLock.Unlock()
			if found && isAssetUnchanged(cacheData, asset) {
				gitCachelog.Info(fmt.Sprintf("Git data retrieved from cache. The data is associated with gitRelease containing: %v", path))
				cacheData.lastUsed = time.Now()
				return cacheData.data, nil
			}

			// The asset is being read for the first time or it was modified and is being read again.
			indexBytes, err := downloadReleaseAsset(gclient, gitRelease, asset)
			if err != nil {
				return nil, err
			}

			// Add downloaded data to cache if the data needed for caching is present.
			gitCacheLock.Lock()
			if asset.GetID() != 0 && (asset.GetCreatedAt() != github.Timestamp{}) && (asset.GetSize() != 0) {
				startPurgeTicker.Do(func() {
					cutils.ScheduleWork(gitTickerDuration, gitCachelog, gitPurgeCache, gitPurgeDuration)
				})
				gitCache[path] = gitCacheData{assetId: asset.GetID(), creationTime: asset.GetCreatedAt().Time, size: asset.GetSize(), data: indexBytes, lastUsed: time.Now()}
				gitCachelog.Info(fmt.Sprintf("Git data cached. The data is associated with gitRelease containing: %v", path))
			} else {
				delete(gitCache, path)
			}
			gitCacheLock.Unlock()

			break
		}
	}

	return indexBytes, nil
}

// Downloads a release asset.
func downloadReleaseAsset(gclient *github.Client, gitRelease kabanerov1alpha2.GitReleaseInfo, asset github.ReleaseAsset) ([]byte, error) {
	// The asset is being read for the first time or was modified.
	reader, _, err := gclient.Repositories.DownloadReleaseAsset(context.Background(), gitRelease.Organization, gitRelease.Project, asset.GetID(), http.DefaultClient)
	if err != nil {
		return nil, fmt.Errorf("Unable to download release asset %v. Configured GitRelease data: %v. Error: %v", gitRelease.AssetName, gitRelease, err)
	}
	defer reader.Close()

	indexBytes, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("Unable to read downloaded asset %v from request. Configured GitRelease data: %v. Error: %v", gitRelease.AssetName, gitRelease, err))
	}
	return indexBytes, nil
}

// Returns true if there is indication that the asset is unchanged. False, otherwise.
func isAssetUnchanged(cacheData gitCacheData, asset github.ReleaseAsset) bool {
	unchanged := (cacheData.assetId == asset.GetID()) &&
		(cacheData.creationTime.Equal(asset.GetCreatedAt().Time)) &&
		(cacheData.size == asset.GetSize())
	return unchanged
}

// Purges the git cache. This function is scheduled to execute by a timer scheduler.
func gitPurgeCache(localPurgeDuration time.Duration) {
	cacheLock.Lock()
	defer cacheLock.Unlock()
	for key, _ := range gitCache {
		if time.Since(gitCache[key].lastUsed) > localPurgeDuration {
			gitCachelog.Info("Purging Git cache entry: " + key)
			delete(gitCache, key)
		}
	}
}
