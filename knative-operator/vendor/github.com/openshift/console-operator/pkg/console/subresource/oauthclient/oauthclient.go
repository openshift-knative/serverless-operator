package oauthclient

import (
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	oauthv1 "github.com/openshift/api/oauth/v1"
	oauthclient "github.com/openshift/client-go/oauth/clientset/versioned/typed/oauth/v1"
	"github.com/openshift/library-go/pkg/operator/resource/resourcemerge"

	"github.com/openshift/console-operator/pkg/api"
	"github.com/openshift/console-operator/pkg/console/subresource/util"
	"github.com/openshift/console-operator/pkg/crypto"
)

// TODO: ApplyOauth should be a generic Apply that could be used for any oauth-client
// - should look like resourceapply.ApplyService and the other Apply funcs
//   once its in a trustworthy state, PR to library-go so it can live with
//   the other Apply funcs
func CustomApplyOAuth(client oauthclient.OAuthClientsGetter, required *oauthv1.OAuthClient) (*oauthv1.OAuthClient, bool, error) {
	existing, err := client.OAuthClients().Get(required.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		actual, err := client.OAuthClients().Create(required)
		return actual, true, err
	}
	if err != nil {
		return nil, false, err
	}
	// TODO: if this is going to be PR'd to library-go, we have to handle all of these fields :/
	// Unfortunately data is all top level so its a little more
	// tedious to manually copy things over
	modified := resourcemerge.BoolPtr(false)
	resourcemerge.EnsureObjectMeta(modified, &existing.ObjectMeta, required.ObjectMeta)
	// at present, we only care about these two fields. this is NOT generic to all oauth clients
	secretSame := equality.Semantic.DeepEqual(existing.Secret, required.Secret)
	redirectsSame := equality.Semantic.DeepEqual(existing.RedirectURIs, required.RedirectURIs)
	// nothing changed, so don't update
	if secretSame && redirectsSame && !*modified {
		// per ApplyService, etc, if nothing changed, return nil.
		return nil, false, nil
	}
	existing.Secret = required.Secret
	// existing.AdditionalSecrets = required.AdditionalSecrets
	// existing.RespondWithChallenges = required.RespondWithChallenges
	existing.RedirectURIs = required.RedirectURIs
	// existing.GrantMethod = required.GrantMethod
	// existing.ScopeRestrictions = required.ScopeRestrictions
	// existing.AccessTokenMaxAgeSeconds = required.AccessTokenMaxAgeSeconds
	// existing.AccessTokenInactivityTimeoutSeconds = required.AccessTokenInactivityTimeoutSeconds
	actual, err := client.OAuthClients().Update(existing)
	return actual, true, err
}

// registers the console on the oauth client as a valid application
func RegisterConsoleToOAuthClient(client *oauthv1.OAuthClient, host string, randomBits string) *oauthv1.OAuthClient {
	SetRedirectURI(client, host)
	// client.Secret = randomBits
	SetSecretString(client, randomBits)
	return client
}

// for ManagementState.Removed
// Console does not have create/delete priviledges on oauth clients, only update
func DeRegisterConsoleFromOAuthClient(client *oauthv1.OAuthClient) *oauthv1.OAuthClient {
	client.RedirectURIs = []string{}
	// changing the string to anything else will invalidate the client
	client.Secret = crypto.Random256BitsString()
	return client
}

func DefaultOauthClient() *oauthv1.OAuthClient {
	return Stub()
}

func Stub() *oauthv1.OAuthClient {
	// we cannot set an ownerRef on the OAuthClient as it is cluster scoped
	return &oauthv1.OAuthClient{
		ObjectMeta: metav1.ObjectMeta{
			Name: api.OAuthClientName,
		},
	}
}

func GetSecretString(client *oauthv1.OAuthClient) string {
	return client.Secret
}

func SetSecretString(client *oauthv1.OAuthClient, randomBits string) *oauthv1.OAuthClient {
	client.Secret = string(randomBits)
	return client
}

// we are the only application for this client
// in the future we may accept multiple routes
// for now, we can clobber the slice & reset the entire thing
func SetRedirectURI(client *oauthv1.OAuthClient, host string) *oauthv1.OAuthClient {
	client.RedirectURIs = []string{}
	client.RedirectURIs = append(client.RedirectURIs, util.HTTPS(host)+"/auth/callback")
	return client
}
