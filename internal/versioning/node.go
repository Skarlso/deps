package versioning

// Node.js ships a new major every six months: even majors become LTS, odd ones are Current
// releases that are supported for half a year and never go LTS. Renovate's node versioning
// reports an odd major as unstable, so its default `ignoreUnstable` holds such a release
// back (`workarounds:typesNodeVersioning`, `workarounds:nodeDockerVersioning` of
// `config:recommended`). deps' node scheme is the docker scheme (the tag shapes are the
// same: `24`, `24.11.0`, `v24.11.0`, `24-alpine`) with that rule on top.

// nodeCompat is dockerCompat plus the LTS rule: a candidate on an odd major is only
// compatible with a version already on that same major, so `24` stays on even majors while
// `25.1.0` still moves to `25.2.0`.
func nodeCompat(cur, next dockerTag) string {
	if why := dockerCompat(cur, next); why != "" {
		return why
	}
	if len(next.parts) > 0 && next.parts[0]%2 == 1 && (len(cur.parts) == 0 || cur.parts[0] != next.parts[0]) {
		return "unstable"
	}
	return ""
}
