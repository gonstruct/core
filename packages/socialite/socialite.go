// Package socialite is the provider-agnostic half of signing in with somebody
// else's account: the OAuth 2.0 authorization code flow, with PKCE.
//
// It reads like its Laravel namesake on purpose. A controller does two things:
//
//	return socialite.Driver(context, "aicmp").Redirect(redirectTo)
//	user, err := socialite.Driver(context, "aicmp").User()
//
// and shapes the request in between when it needs to:
//
//	socialite.Driver(context, "google").
//		Scopes("https://www.googleapis.com/auth/drive.readonly").
//		With(map[string]string{"login_hint": address}).
//		Redirect("/settings/integrations")
//
// What it does that Socialite does not:
//
//   - PKCE (RFC 7636) is on unless a provider says it cannot, rather than off
//     unless somebody remembers. OAuth 2.1 requires it of every client.
//   - A provider that refuses in the redirect (RFC 6749 section 4.1.2.1) is an
//     AuthorizationError, so a cancelled sign-in is not reported as a broken
//     one.
//   - The iss parameter (RFC 9207) is checked for a provider that sends it,
//     which is what defeats a mix-up between two authorization servers.
//   - State is compared in constant time, the handshake carries its own expiry,
//     and a nonce is issued whenever the openid scope is asked for.
//   - Granted scopes come back on the User, because RFC 6749 section 5.1 lets
//     them differ from the ones requested.
//
// It deliberately stops at the identity. What an application does with the
// resolved user -- sign them in, attach them to a workspace, refuse them --
// differs far more between features than providers differ from each other.
package socialite

import (
	"maps"
	"sync"

	"github.com/gin-gonic/gin"
)

// Drivers maps a name to a provider constructor. It holds constructors rather
// than instances so each request reads the configuration as it is now: a
// provider built once at boot would keep whatever credentials existed then,
// which is wrong the moment anything reloads them, and untestable besides.
type Drivers map[string]func() Provider

// Configuration is what the application registers once at boot: how to seal a
// handshake, what the cookie looks like, and which providers exist.
type Configuration struct {
	Sealer  Sealer
	Cookie  CookieOptions
	Drivers Drivers
}

var (
	registered Configuration
	mutex      sync.RWMutex
)

// Register binds the drivers. Calling it twice replaces the previous set, so a
// test can stand its own providers up.
func Register(configuration Configuration) {
	mutex.Lock()
	defer mutex.Unlock()

	registered = configuration

	if registered.Drivers == nil {
		registered.Drivers = Drivers{}
	} else {
		registered.Drivers = maps.Clone(configuration.Drivers)
	}
}

// Extend adds one driver to the registered set, for a provider that is not part
// of the application's own configuration. It is Socialite::extend.
func Extend(name string, construct func() Provider) {
	mutex.Lock()
	defer mutex.Unlock()

	if registered.Drivers == nil {
		registered.Drivers = Drivers{}
	}

	registered.Drivers[name] = construct
}

// Driver binds a provider to this request. An unknown name is not a panic: it
// surfaces as ErrUnknownDriver from Redirect or User, so a typo in a route
// fails as a response rather than as a crash.
func Driver(context *gin.Context, name string) *Flow {
	mutex.RLock()
	defer mutex.RUnlock()

	bound := &Flow{
		context: context,
		name:    name,
		sealer:  registered.Sealer,
		cookie:  registered.Cookie,
	}

	if construct, ok := registered.Drivers[name]; ok {
		bound.provider = construct()
	}

	return bound
}
