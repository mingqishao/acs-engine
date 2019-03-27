package acsengine

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/Azure/azure-sdk-for-go/arm/resources/subscriptions"
	"github.com/Azure/go-autorest/autorest/azure"
	log "github.com/sirupsen/logrus"
)

// GetTenantID figures out the AAD tenant ID of the subscription by making an
// unauthenticated request to the Get Subscription Details endpoint and parses
// the value from WWW-Authenticate header.
func GetTenantID(env azure.Environment, subscriptionID string) (string, error) {
	const hdrKey = "WWW-Authenticate"
	c := subscriptions.NewGroupClient()
	c.BaseURI = env.ResourceManagerEndpoint

	log.Debugf("Resolving tenantID for subscriptionID: %s", subscriptionID)

	// we expect this request to fail (err != nil), but we are only interested
	// in headers, so surface the error if the Response is not present (i.e.
	// network error etc)
	subs, err := c.Get(subscriptionID)
	if subs.Response.Response == nil {
		log.Errorf("Request failed: %v", err)
		return "", fmt.Errorf("Request failed: %v", err)
	}

	// Expecting 401 StatusUnauthorized here, just read the header
	if subs.StatusCode != http.StatusUnauthorized {
		log.Errorf("Unexpected response from Get Subscription: %v", subs.StatusCode)
		return "", fmt.Errorf("Unexpected response from Get Subscription: %v", subs.StatusCode)
	}
	hdr := subs.Header.Get(hdrKey)
	if hdr == "" {
		log.Errorf("Header %v not found in Get Subscription response", hdrKey)
		return "", fmt.Errorf("Header %v not found in Get Subscription response", hdrKey)
	}

	// Example value for hdr:
	//   Bearer authorization_uri="https://login.windows.net/996fe9d1-6171-40aa-945b-4c64b63bf655", error="invalid_token", error_description="The authentication failed because of missing 'Authorization' header."
	r := regexp.MustCompile(`authorization_uri=".*/([0-9a-f\-]+)"`)
	m := r.FindStringSubmatch(hdr)
	if m == nil {
		log.Errorf("Could not find the tenant ID in header: %s %q", hdrKey, hdr)
		return "", fmt.Errorf("Could not find the tenant ID in header: %s %q", hdrKey, hdr)
	}
	return m[1], nil
}
