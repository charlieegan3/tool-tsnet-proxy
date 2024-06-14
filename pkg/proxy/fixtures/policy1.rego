package authz

import rego.v1

default allow := false

allow if {
    print(input)
    input.headers["X-Test"]
}