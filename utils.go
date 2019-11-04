package surfkit

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/compute/metadata"
)

// NewAuthenticateableRequest prepares a Request object to be executed
// on the given URL with the given method. If a local metadata service is available,
// it retrieves a Bearer token from it and attaches this token to the Head of the newly
// created request object.
//
// Availability of a local metadata service, meaning the requirement to authenticate the request,
// is determined by the request url. It is expected that secure HTTP indicates this requirement.
//
// GCP metadata server can be a little shaky at times so if a token cannot be
// retrieved from the local metadata service, it will try again until a token
// is successfully retrieved or a maximum backoff time of 16 seconds is reached.
//
// For details on how the backoff works
// check https://cloud.google.com/storage/docs/exponential-backoff
func NewAuthenticateableRequest(method, url string, body io.Reader) (*http.Request, error) {

	if !strings.HasPrefix(url, "https") {
		return http.NewRequest(method, url, body)
	}

	var authToken string
	var err error

	maxbackoff := time.Now().Add(16 * time.Second)
	backoffIter := 1

	for {
		// query the id_token with ?audience as the serviceURL
		tokenURL := fmt.Sprintf("/instance/service-accounts/default/identity?audience=%s", url)
		authToken, err = metadata.Get(tokenURL)

		if err == nil {
			break

		} else {
			switch t := err.(type) {

			// Metadata errors. There is a metadata service but it seems to have problems
			case *metadata.Error:
				log.Printf("AuthenticatedRequest: Failed to query metadata. Status: %d, Message: %s (%+v)", t.Code, t.Message, err)

			// Connection errors. Likely there is no metadata service so we skip authentication.
			case net.Error:
				log.Printf("AuthenticatedRequest: Failed to query metadata. Connection problem (%+v)", err)

			// All other errors
			default:
				return nil, fmt.Errorf("AuthenticatedRequest: Failed to query metadata: (%+v)", err)
			}
		}

		if time.Now().After(maxbackoff) {
			return nil, fmt.Errorf("AuthenticatedRequest: Failed to query metadata. Max backoff time passed")
		}

		waitTime := (backoffIter * 1000) + rand.New(rand.NewSource(time.Now().UnixNano())).Intn(500)
		backoffIter = backoffIter * 2

		log.Printf("AuthenticatedRequest: Backing off for %d", waitTime)
		time.Sleep(time.Duration(waitTime) * time.Millisecond)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", authToken))
	return req, nil

}

// DoAuthenticateableRequest creates an authenticatable http Request and executes it.
// See NewAuthenticateableRequest for more details
func DoAuthenticateableRequest(method, url string, body io.Reader) (*http.Response, error) {
	var err error

	req, err := NewAuthenticateableRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	return client.Do(req)
}
