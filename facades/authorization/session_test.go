package authorization_test

import (
	"testing"

	"github.com/gonstruct/core/facades/authorization"
)

func TestOnlyLimitsASessionToWhatItNames(t *testing.T) {
	authorization.Register(func() map[string]authorization.RolesWithPermissions {
		return map[string]authorization.RolesWithPermissions{
			"only-test": {"admin": {"read": true, "write": true, "delete": true}},
		}
	})

	session := authorization.NewSession[string]("only-test", "admin")
	if !session.CanAll("read", "write", "delete") {
		t.Fatal("an unlimited admin cannot do what the role allows")
	}

	limited := session.Only("read", "write", "publish")
	if !limited.Can("read") || !limited.Can("write") {
		t.Error("a limited session lost what it was left")
	}
	if limited.Can("delete") || !limited.Cannot("delete") {
		t.Error("a limited session kept what it was not given")
	}
	if limited.Can("publish") {
		t.Error("a limit gave what the role does not allow")
	}

	narrower := limited.Only("write", "delete")
	if narrower.Can("read") || narrower.Can("delete") || !narrower.Can("write") {
		t.Error("limiting twice did not keep only what both name")
	}

	if session.Only().CanAny("read", "write", "delete") {
		t.Error("a session limited to nothing can do something")
	}
}
