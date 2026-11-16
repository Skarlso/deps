package bot

import (
	"context"
	"slices"
	"strings"
	"testing"

	"git.jakob-moeller.cloud/jakobmoellerdev/deps/internal/dep"
	"git.jakob-moeller.cloud/jakobmoellerdev/deps/internal/engine"
	"git.jakob-moeller.cloud/jakobmoellerdev/deps/internal/platform"
	"git.jakob-moeller.cloud/jakobmoellerdev/deps/internal/schedule"
)

func TestRebaseCheckbox(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		ticked     bool
	}{
		{"unticked", "x\n" + rebaseCheckbox + "\n", false},
		{"ticked", "x\n- [x] <!-- deps:rebase --> If you want to rebase/retry this PR, check this box\n", true},
		{"ticked upper case", "- [X] <!-- deps:rebase --> anything\n", true},
		{"star bullet, indented", "  * [x] <!-- deps:rebase -->\n", true},
		{"dashboard checkbox is not the PR's", "- [x] <!-- deps:rebase=deps/a --> a\n", false},
		{"not at line start", "text - [x] <!-- deps:rebase -->\n", false},
		{"none", "x\n<!-- deps:fingerprint=abc -->\n", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := rebaseTicked(tc.body); got != tc.ticked {
				t.Errorf("rebaseTicked = %v, want %v", got, tc.ticked)
			}
			un := untickRebase(tc.body)
			if rebaseTicked(un) {
				t.Errorf("untickRebase left it ticked: %q", un)
			}
			if !tc.ticked && un != tc.body {
				t.Errorf("untickRebase changed %q to %q", tc.body, un)
			}
			if tc.ticked && len(un) != len(tc.body) {
				t.Errorf("untickRebase must only clear the box: %q -> %q", tc.body, un)
			}
		})
	}
}

// The checkbox is in every body, unticked, above the markers; ticking it changes neither
// fingerprint, so it never reads as new content.
func TestPRBodyRebaseCheckbox(t *testing.T) {
	_, res := fixture(t)
	body := prBody(t.Context(), Options{}, engine.Group{Updates: res.Updates[:1]}, "abc", "def")
	if strings.Count(body, rebaseCheckbox) != 1 || rebaseTicked(body) {
		t.Fatalf("body lacks one unticked checkbox:\n%s", body)
	}
	if !strings.HasSuffix(body, "\n---\n\n"+rebaseCheckbox+"\n\n<!-- deps:fingerprint=abc -->\n<!-- deps:versions=a%2Fx=v2 -->\n<!-- deps:input=def -->\n") {
		t.Errorf("checkbox not above the markers:\n%s", body)
	}
	ticked := strings.Replace(body, "- [ ] <!-- deps:rebase -->", "- [x] <!-- deps:rebase -->", 1)
	if !rebaseTicked(ticked) || bodyFingerprint(ticked) != "abc" || bodyInput(ticked) != "def" || untickRebase(ticked) != body {
		t.Errorf("ticked body:\n%s", ticked)
	}
}

// quotingNotes answers release notes that quote the body of another deps pull request, as a
// dependency whose changelog shows one does (deps' own releases, for one): the markers land
// in the body verbatim, above the PR's own.
type quotingNotes struct{ fakeNotes }

func (*quotingNotes) Section(_ context.Context, ups []dep.Update) string {
	return "### Release notes\n\n<details>\n<summary>Release notes: " + ups[0].Dep.Name + "</summary>\n\n" +
		"The pull request bodies now end with:\n\n" +
		"- [x] <!-- deps:rebase --> If you want to rebase/retry this PR, check this box\n\n" +
		"<!-- deps:fingerprint=0000000000000000 -->\n" +
		"<!-- deps:versions=quoted%2Fdep=v9.9.9 -->\n" +
		"<!-- deps:input=1111111111111111 -->\n\n</details>\n"
}

// Markers above the PR's own, quoted by release notes or written by the configured header
// and footer, are not the PR's: read instead of the real ones they would rebuild and
// force-push the PR on every run (fingerprint and input), suppress it against versions it
// never proposed (versions) or rebase it unasked (the checkbox).
func TestBodyMarkersQuotedAbove(t *testing.T) {
	u := dep.Update{Dep: dep.Dep{File: "go.mod", Line: 3, Name: "a/x", Version: "v1.0.0"}, Type: dep.Major, NewVersion: "v2.0.0"}
	grp := engine.Group{
		Updates: []dep.Update{u},
		Header:  "<!-- deps:fingerprint=2222222222222222 -->",
		Footer:  "<!-- deps:input=3333333333333333 -->",
	}
	body := prBody(t.Context(), Options{Notes: &quotingNotes{}}, grp, "abc123", "def456")
	if got := bodyFingerprint(body); got != "abc123" {
		t.Errorf("bodyFingerprint = %q, want %q:\n%s", got, "abc123", body)
	}
	if got := bodyInput(body); got != "def456" {
		t.Errorf("bodyInput = %q, want %q:\n%s", got, "def456", body)
	}
	vs := bodyVersions(body)
	if len(vs) != 1 || len(vs["a/x"]) != 1 || vs["a/x"][0] != (target{version: "v2.0.0"}) {
		t.Errorf("bodyVersions = %v, want a/x=v2.0.0 only:\n%s", vs, body)
	}
	if rebaseTicked(body) {
		t.Errorf("a quoted ticked checkbox reads as a rebase request:\n%s", body)
	}
	// Ticking the PR's own box is a rebase request; unticking it after the rebase leaves
	// the quoted one alone, so the body is otherwise unchanged.
	ticked := strings.Replace(body, rebaseCheckbox, strings.Replace(rebaseCheckbox, "[ ]", "[x]", 1), 1)
	if !rebaseTicked(ticked) {
		t.Errorf("ticked checkbox not read:\n%s", ticked)
	}
	if got := untickRebase(ticked); got != body {
		t.Errorf("untickRebase = %q, want the original body", got)
	}
	// withInput replaces the PR's marker, not the quoted one.
	if got := withInput(body, "fed654"); bodyInput(got) != "fed654" ||
		!strings.Contains(got, "<!-- deps:input=1111111111111111 -->") {
		t.Errorf("withInput rewrote the quoted marker:\n%s", got)
	}
}

// In a workflow run the body links it above the rule, the link text escaped; the checkbox
// and the markers are unaffected.
func TestPRBodyRunLink(t *testing.T) {
	_, res := fixture(t)
	o := Options{Run: platform.WorkflowRun{Workflow: "deps [nightly]", Number: "42", URL: "https://github.com/o/r/actions/runs/9/attempts/2"}}
	body := prBody(t.Context(), o, engine.Group{Updates: res.Updates[:1]}, "abc", "")
	want := "\n\nWritten by deps in [deps \\[nightly\\] #42](https://github.com/o/r/actions/runs/9/attempts/2).\n\n---\n\n" + rebaseCheckbox + "\n\n<!-- deps:fingerprint=abc -->\n"
	if !strings.Contains(body, want) || bodyFingerprint(body) != "abc" || rebaseTicked(body) {
		t.Errorf("body lacks %q:\n%s", want, body)
	}
	o.Run = platform.WorkflowRun{URL: "https://g.example/o/r/actions/runs/9"}
	if body := prBody(t.Context(), o, engine.Group{Updates: res.Updates[:1]}, "abc", ""); !strings.Contains(body, "\nWritten by deps in [workflow run](https://g.example/o/r/actions/runs/9).\n") {
		t.Errorf("body without workflow name and number:\n%s", body)
	}
}

// prRequestBody is an open PR body with content fingerprint fp and the checkbox ticked or
// not.
func prRequestBody(fp string, ticked bool) string {
	box := rebaseCheckbox
	if ticked {
		box = strings.Replace(box, "[ ]", "[x]", 1)
	}
	return "x\n\n---\n\n" + box + "\n\n<!-- deps:fingerprint=" + fp + " -->\n"
}

// TestRunPRRequests drives Run with the PR's rebase checkbox, rebaseLabel and
// stopUpdatingLabel set on open PRs.
func TestRunPRRequests(t *testing.T) {
	yes := true
	weekly, err := schedule.Parse([]string{"0 0 * * 0"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	const stale = "0000000000000000" // a fingerprint of other content
	const gone, old = "deps/gomod/gone", "deps/gomod/old"
	tests := []struct {
		name          string
		opts          Options
		outOfSchedule bool // every update has a weekly schedule; the run is triggered by another cron
		author        string
		pr            platform.PR // the open PR of branchA; Body "" = current content, unticked
		ticked        bool        // the PR's checkbox is ticked (Body "")
		extraOpen     []platform.PR
		fail          map[string]error
		wantStatus    map[string]string
		wantPushed    []string
		wantUnlabeled []string
		wantClosed    []int
		wantErr       string
		wantBody      string // "" = no UpdatePR of #1; else a substring of its body
	}{
		{
			name:          "ticked checkbox rebases out of schedule and discards foreign commits",
			outOfSchedule: true, author: "human@example.invalid", ticked: true,
			wantStatus: map[string]string{branchA: StatusRebased, branchB: StatusSkipSchedule},
			wantPushed: []string{branchA},
			wantBody:   "\n" + rebaseCheckbox + "\n",
		},
		{
			name:          "unticked checkbox out of schedule stays up to date",
			outOfSchedule: true, author: "human@example.invalid",
			opts:       Options{UpdateNotScheduled: true},
			wantStatus: map[string]string{branchA: StatusUpToDate, branchB: StatusSkipSchedule},
			wantBody:   "<!-- deps:input=",
		},
		{
			name:          "rebase label rebases and is removed",
			pr:            platform.PR{Labels: []string{"dependencies", "Rebase"}},
			wantStatus:    map[string]string{branchA: StatusRebased, branchB: StatusCreated},
			wantPushed:    []string{branchA, branchB},
			wantUnlabeled: []string{"1:rebase"},
			wantBody:      "<!-- deps:input=",
		},
		{
			name:          "configured rebase label",
			opts:          Options{RebaseLabel: "deps-rebase"},
			pr:            platform.PR{Labels: []string{"deps-rebase"}},
			wantStatus:    map[string]string{branchA: StatusRebased, branchB: StatusCreated},
			wantPushed:    []string{branchA, branchB},
			wantUnlabeled: []string{"1:deps-rebase"},
			wantBody:      "<!-- deps:input=",
		},
		{
			name:       "default rebase label is ignored when another is configured",
			opts:       Options{RebaseLabel: "deps-rebase"},
			pr:         platform.PR{Labels: []string{"rebase"}},
			wantStatus: map[string]string{branchA: StatusUpToDate, branchB: StatusCreated},
			wantPushed: []string{branchB},
			wantBody:   "<!-- deps:input=",
		},
		{
			name:          "rebase label on changed content is removed after the update",
			pr:            platform.PR{Body: prRequestBody(stale, true), Labels: []string{"rebase"}},
			wantStatus:    map[string]string{branchA: StatusUpdated, branchB: StatusCreated},
			wantPushed:    []string{branchA, branchB},
			wantUnlabeled: []string{"1:rebase"},
			wantBody:      "\n" + rebaseCheckbox + "\n",
		},
		{
			name:       "failed label removal is reported",
			pr:         platform.PR{Labels: []string{"rebase"}},
			fail:       map[string]error{"RemovePRLabel": errBoom},
			wantStatus: map[string]string{branchA: StatusRebased, branchB: StatusCreated},
			wantPushed: []string{branchA, branchB},
			wantErr:    `remove label "rebase" from #1: boom`,
			wantBody:   "<!-- deps:input=",
		},
		{
			name:       "dry run would rebase and writes nothing",
			opts:       Options{DryRun: true},
			ticked:     true,
			pr:         platform.PR{Labels: []string{"rebase"}},
			wantStatus: map[string]string{branchA: StatusWouldRebase, branchB: StatusWouldCreate},
		},
		{
			name:       "stop-updating label leaves a changed PR alone, also when a rebase is requested",
			pr:         platform.PR{Body: prRequestBody(stale, true), Labels: []string{"rebase", "Stop-Updating"}},
			wantStatus: map[string]string{branchA: StatusSkipStopped, branchB: StatusCreated},
			wantPushed: []string{branchB},
		},
		{
			name: "configured stop label",
			opts: Options{StopUpdatingLabel: "hands-off"},
			pr:   platform.PR{Body: prRequestBody(stale, false), Labels: []string{"hands-off"}},
			extraOpen: []platform.PR{
				{Number: 9, Head: gone, Labels: []string{"stop-updating"}}, // not the configured label: closed
			},
			wantStatus: map[string]string{branchA: StatusSkipStopped, branchB: StatusCreated, gone: StatusClosed},
			wantPushed: []string{branchB},
			wantClosed: []int{9},
		},
		{
			name: "stop-updating label keeps an obsolete PR open",
			extraOpen: []platform.PR{
				{Number: 9, Head: gone, Labels: []string{"stop-updating"}},
				{Number: 10, Head: old},
			},
			wantStatus: map[string]string{branchA: StatusUpToDate, branchB: StatusCreated, gone: StatusSkipStopped, old: StatusClosed},
			wantPushed: []string{branchB},
			wantClosed: []int{10},
			wantBody:   "<!-- deps:input=",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, res := fixture(t)
			fpA := fingerprintOf(t, root, res, "a/x")
			o := tt.opts
			o.AuthorEmail, o.Managers, o.MaxNewPRs = email, managers, 10
			if tt.outOfSchedule {
				for i := range res.Updates {
					res.Updates[i].Schedule = weekly
				}
				o.Trigger = schedule.Trigger{Event: schedule.EventSchedule, Cron: "0 0 * * *"}
			}
			pr := tt.pr
			pr.Number, pr.Head, pr.Title, pr.URL, pr.Mergeable = 1, branchA, titleA, "u1", &yes
			if pr.Body == "" {
				pr.Body = prRequestBody(fpA, tt.ticked)
			}
			pc := &fakePC{open: append([]platform.PR{pr}, tt.extraOpen...), fail: tt.fail}
			author := tt.author
			if author == "" {
				author = email
			}
			branches := map[string]string{"refs/heads/" + branchA: "a"}
			for _, p := range tt.extraOpen {
				branches["refs/heads/"+p.Head] = "x"
			}
			g := &fakeGit{branches: branches, authors: map[string]string{branchA: author}}
			acts, errs := Run(t.Context(), o, []Base{{Root: root, Res: res, Git: g}}, nil, pc)
			switch {
			case tt.wantErr == "" && len(errs) > 0:
				t.Fatalf("errs = %v", errs)
			case tt.wantErr != "" && (len(errs) != 1 || !strings.Contains(errs[0].Error(), tt.wantErr)):
				t.Fatalf("errs = %v, want %q", errs, tt.wantErr)
			}
			for b, want := range tt.wantStatus {
				if got := statusOf(acts, b); got != want {
					t.Errorf("%s: status %q, want %q", b, got, want)
				}
			}
			if len(acts) != len(tt.wantStatus) {
				t.Errorf("actions %+v, want %v", acts, tt.wantStatus)
			}
			if !slices.Equal(g.pushed, tt.wantPushed) {
				t.Errorf("pushed %v, want %v", g.pushed, tt.wantPushed)
			}
			if !slices.Equal(pc.unlabeled, tt.wantUnlabeled) {
				t.Errorf("labels removed %v, want %v", pc.unlabeled, tt.wantUnlabeled)
			}
			if !slices.Equal(pc.closedNow, tt.wantClosed) {
				t.Errorf("closed %v, want %v", pc.closedNow, tt.wantClosed)
			}
			body, written := pc.prBodies[1]
			switch {
			case tt.wantBody == "" && written:
				t.Errorf("PR #1 body written:\n%s", body)
			case tt.wantBody != "" && (!strings.Contains(body, tt.wantBody) || rebaseTicked(body)):
				t.Errorf("PR #1 body lacks %q or is ticked:\n%s", tt.wantBody, body)
			case tt.wantBody != "" && pc.open[0].Body == prRequestBody(fpA, true) && body != withInput(prRequestBody(fpA, false), bodyInput(body)):
				// A rebase only unticks the box and records the input fingerprint.
				t.Errorf("rebased PR body:\n%s", body)
			}
		})
	}
}

func TestDashboardStoppedPR(t *testing.T) {
	root, res := dashFixture(t)
	open := []platform.PR{{Number: 1, Head: branchA, Title: titleA, URL: "u1", Labels: []string{"stop-updating"}, Body: prRequestBody("0000000000000000", false)}}
	pc := &fakePC{open: open}
	acts, errs, _ := dashRun(t, Options{}, res, root, pc, &fakeGit{})
	if len(errs) > 0 || statusOf(acts, branchA) != StatusSkipStopped {
		t.Fatalf("%v %+v", errs, acts)
	}
	body := pc.createdIssues[0].Body
	want := "## Ignored or blocked\n\n- " + titleA + " ([#1](u1) has the `stop-updating` label; deps leaves it alone until the label is removed)\n"
	if !strings.Contains(body, want) || strings.Contains(body, "deps:rebase="+branchA) {
		t.Errorf("body lacks %q (and must not offer a rebase):\n%s", want, body)
	}
}
