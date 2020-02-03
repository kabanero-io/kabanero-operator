package stack

import (
	"testing"

	"bytes"
	"net/http"
	"net/http/httptest"
)

const theResponse = "The response."
const theResponse2 = "The response2."

// HTTP handler that lets us know if the caller asked for the etag.
type CacheHandler struct {
	etag string
	cacheHits *int32
}

func (ch CacheHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// Check to see if the request specified the If-None-Match header
	etagHeader := req.Header.Get("If-None-Match")
	if etagHeader == ch.etag {
		// Indicate the resource has not changed.
		rw.WriteHeader(http.StatusNotModified)
		*(ch.cacheHits) += 1
	} else {
		// Just write the response
		rw.Header().Add("ETag", ch.etag)
		rw.Header().Add("Date", "GarbageDate")
		rw.Write([]byte(theResponse))
	}
}

// Show that the client is sending the correct etag on a subsequent request.
func TestCachePage(t *testing.T) {
	var cacheHits int32 = 0
	handler := CacheHandler{etag: "ABCDE", cacheHits: &cacheHits}
	server := httptest.NewServer(handler)
	defer server.Close()

	// Get the page twice... the first time should not cache, the second should cache.
	data, err := getFromCache(server.URL, false)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Compare([]byte(theResponse), data) != 0 {
		t.Fatal("Response 1 not correct")
	}

	data, err = getFromCache(server.URL, false)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Compare([]byte(theResponse), data) != 0 {
		t.Fatal("Response 2 not correct")
	}

	// Make sure that the cache hit one time.
	if cacheHits != 1 {
		t.Fatalf("Wrong number of cache hits: %v", cacheHits)
	}
}

// HTTP handler that lets us know if the caller asked for the etag.
type CacheChangeHandler struct {
	etag1, etag2 string
	cacheHits *int32
}

func (ch CacheChangeHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// Check to see if the request specified the If-None-Match header
	etagHeader := req.Header.Get("If-None-Match")
	if etagHeader == ch.etag1 {
		// Got back the first etag, change it up and return the second.
		rw.Header().Add("ETag", ch.etag2)
		rw.Header().Add("Date", "GarbageDate")
		rw.Write([]byte(theResponse2))
	} else if etagHeader == ch.etag2 {
		// Indicate the resource has not changed.
		rw.WriteHeader(http.StatusNotModified)
		*(ch.cacheHits) += 1
	} else {
		// Just write the response
		rw.Header().Add("ETag", ch.etag1)
		rw.Header().Add("Date", "GarbageDate")
		rw.Write([]byte(theResponse))
	}
}

// Show that if the server changes the etag, the client will update it.
func TestCacheChangePage(t *testing.T) {
	var cacheHits int32 = 0
	handler := CacheChangeHandler{etag1: "ABCDE", etag2: "EFGHI", cacheHits: &cacheHits}
	server := httptest.NewServer(handler)
	defer server.Close()

	// Get the page thrice... the first time and second time should not cache, the third should cache.
	data, err := getFromCache(server.URL, false)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Compare([]byte(theResponse), data) != 0 {
		t.Fatal("Response 1 not correct")
	}

	data, err = getFromCache(server.URL, false)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Compare([]byte(theResponse2), data) != 0 {
		t.Fatal("Response 2 not correct")
	}

	data, err = getFromCache(server.URL, false)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Compare([]byte(theResponse2), data) != 0 {
		t.Fatal("Response 3 not correct")
	}

	// Make sure that the cache hit one time.
	if cacheHits != 1 {
		t.Fatalf("Wrong number of cache hits: %v", cacheHits)
	}
}

// HTTP handler that does not cache
type NoCacheHandler struct {}

func (ch NoCacheHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	rw.Write([]byte(theResponse))
}

// Show that the cache doesn't care if the server does not send etags
func TestNoCachePage(t *testing.T) {
	handler := NoCacheHandler{}
	server := httptest.NewServer(handler)
	defer server.Close()

	// Get the page twice... 
	data, err := getFromCache(server.URL, false)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Compare([]byte(theResponse), data) != 0 {
		t.Fatal("Response 1 not correct")
	}

	data, err = getFromCache(server.URL, false)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Compare([]byte(theResponse), data) != 0 {
		t.Fatal("Response 2 not correct")
	}
}

// Test that we can purge an entry from the cache successfully.
func TestCachePurge(t *testing.T) {
	var cacheHits int32 = 0
	handler := CacheHandler{etag: "ABCDE", cacheHits: &cacheHits}
	server := httptest.NewServer(handler)
	defer server.Close()

	// Get the page twice... the first time should not cache.
	data, err := getFromCache(server.URL, false)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Compare([]byte(theResponse), data) != 0 {
		t.Fatal("Response 1 not correct")
	}

	// Now purge the cache
	purgeCache(0)

	// Get the page the second time... it should not be cached.
	data, err = getFromCache(server.URL, false)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Compare([]byte(theResponse), data) != 0 {
		t.Fatal("Response 2 not correct")
	}

	// Make sure that the cache did not hit.
	if cacheHits != 0 {
		t.Fatalf("Wrong number of cache hits: %v", cacheHits)
	}
}
