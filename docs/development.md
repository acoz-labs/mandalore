# Development

Run `bin/ci` from the repository root. At bootstrap this checks the managed
solution-plan contract, shell syntax, required documentation, public fixture
hygiene and workflow runner configuration. It does not test an unported runtime.

`bin/container bin/ci` is the container entrypoint. If Docker is unavailable,
use the supported host fallback `bin/ci` and report the route actually verified.
The runtime port must pin Go and test tools before enabling language builds;
do not install an unrecorded latest runtime.

Use synthetic banks and disposable native profiles. Never copy ambient native
authentication, private transcripts or real memory into acceptance fixtures.

## Public CI

Every job uses the explicit `CI_RUNNER=ubuntu-latest` repository variable,
following the public predecessor's pattern. No public fork code goes to private
runners. This is an intentional configuration, not an automatic fallback.
Release/acceptance workflows retain their separate permissions and checks.

Local validation is not independent product acceptance. Record exact source and
artifact identities, actual native results and pending checks in the issue/PR.
