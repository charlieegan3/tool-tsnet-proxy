package authz

import rego.v1

default allow := false

allow if {
	input.headers["X-Test"]
}
