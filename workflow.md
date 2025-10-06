This document outlines the basic workflow you should follow. 
Follow these guidelines to remain in scope, and to push updates to 
the correct places. You'll also find notes on git messages here too. 
Try to keep your commits clean and messages sensible.


## Git Workflow (dev → beta → main)

This gives you a clean, repeatable flow for bugfixes & features, from issue → dev → beta → main, with commit/PR hygiene baked in.

---

### 0) Branch model (ground rules)

* **main**: production-only. Tagged releases live here.
* **beta**: broader testing for select users. Promotion from `dev`.
* **dev**: your daily integration branch. Feature branches merge here first.
* **Never force-push** `dev`, `beta`, or `main`. Protect them in your host (GitHub/GitLab).

---

### 1) Start with an issue (recommended)

Create an issue first; it keeps everything traceable.

**Issue template (minimal):**

* **Title:** concise outcome (e.g., “Fix: SDL audio device initialization stalls”)
* **Problem statement:** what’s broken or missing.
* **Repro steps / expected vs. actual**
* **Acceptance criteria:** bullet list of done-ness checks.
* **Risk & impact:** blast radius if wrong.
* **Scope:** in/out of scope bullets.
* **Links:** logs, screenshots.
* **Estimate/labels/milestone**

On GitHub, use closing keywords later in the PR description: `Fixes #123`, `Closes #123`, etc.

---

### 2) Sync & branch off `dev`

```bash
git fetch origin
git switch dev
git pull --ff-only origin dev

# Name branches: type/issue-short-description
# types: feat, fix, chore, refactor, docs, test, perf, build, ci
git switch -c fix/123-audio-init-stall
```

> Tip: include the issue number (`123`) in the branch name to auto-link.

---

### 3) Do the work (commit early, commit well)

#### Make small, logical commits

* One intent per commit (atomic).
* Commit whenever you complete a testable step.

#### Commit message format (Conventional Commits)

* **Subject line:** `<type>(scope?): <imperative summary>`

  * **≤ 50 chars**, imperative mood, no trailing period.
* **Blank line**
* **Body:** wrap \~72 chars, explain the **why**; reference decisions, tradeoffs.
* **Footer:** “Fixes #123”, “Co-authored-by: …” etc.

**Examples**

```
fix(audio): avoid stall when default output is busy

Detect busy device and retry selection with fallback strategy.
Adds 200ms backoff and logs a single warning.

Fixes #123
```

```
feat(ui): add latency meter overlay

Shows rolling 5s average and peak to aid QA and demos.
```

Optionally sign commits if you use GPG:

```bash
git commit -S -m "fix(audio): avoid stall when default output is busy"
```

---

### 4) Keep your branch up to date

Prefer **rebasing your feature branch onto dev** (since it’s your branch):

```bash
git fetch origin
git rebase origin/dev
# resolve conflicts if any
git push -u origin HEAD --force-with-lease   # ok to force *your* feature branch
```

> Don’t rebase `dev/beta/main`. Only rebase your own feature branches.

---

### 5) Open a Draft PR → `dev`

* Target: **`dev`**
* Title: match your main commit/issue.
* Description: problem, solution, screenshots, **Acceptance criteria**, **Test plan**, **Risks**, **Fixes #123**.
* Convert to “Ready for review” when tests pass.

**CLI (optional with GitHub CLI):**

```bash
gh pr create --fill --base dev --head fix/123-audio-init-stall --draft
```

---

### 6) Review checklist (self or team)

* [ ] PR addresses the issue & acceptance criteria.
* [ ] Unit/integration tests updated/added.
* [ ] Logs are meaningful; no noisy debug left.
* [ ] Config/migrations documented, if any.
* [ ] Security & performance considered.
* [ ] Works on target platforms/devices.

---

### 7) Merge to `dev`

Use a **Squash Merge** into `dev` to keep history tidy (one logical commit per feature):

* **Commit title:** keep the best subject (e.g., `fix(audio): avoid stall when default output is busy`)
* **Body:** keep rationale + “Fixes #123”

This keeps `dev` readable and won’t rewrite it later.

---

### 8) Promote `dev` → `beta`

When `dev` is green and you want wider testing:

```bash
git fetch origin
git switch beta
git pull --ff-only origin beta
git merge --no-ff origin/dev
git push origin beta
```

* Optionally open a PR `dev → beta` if you want review & CI gates.
* Optionally bump a **pre-release version** like `v3.3.0-beta.1` (semver pre-release).

---

### 9) Promote `beta` → `main` (release)

After successful beta validation:

```bash
git fetch origin
git switch main
git pull --ff-only origin main
git merge --no-ff origin/beta
git push origin main
```

Tag & release:

```bash
VERSION=v3.3.0
git tag -a "$VERSION" -m "Release $VERSION"
git push origin "$VERSION"
```

Create a GitHub Release (notes can be auto-generated if you follow Conventional Commits). Attach artifacts if needed.

---

### 10) Hotfix flow (urgent prod bug)

* Branch from **`main`**:

  ```bash
  git fetch origin
  git switch main && git pull --ff-only
  git switch -c hotfix/456-panic-on-empty-config
  # fix, commit, PR to main
  ```
* Merge to **`main`**, tag a patch (e.g., `v3.3.1`).
* **Back-merge** to `beta` and `dev` to keep them in sync:

  ```bash
  git switch beta && git pull --ff-only && git merge --no-ff origin/main && git push
  git switch dev  && git pull --ff-only && git merge --no-ff origin/main && git push
  ```

---

### 11) Issue hygiene

* Use labels: `bug`, `feat`, `perf`, `ui`, `security`, `priority:high`, `area:audio`, etc.
* Keep issues small & actionable; split large ones.
* Close via PR with `Fixes #id`.
* Add **milestones** for releases (`v3.3.0`, `v3.4.0`).
* Convert recurring chores to separate issues; don’t hide them in PRs.

---

### 12) Handy commands & aliases

```bash
# Update locals from remote, fast-forward only (safe)
git fetch origin
git switch dev && git pull --ff-only origin dev
git switch beta && git pull --ff-only origin beta
git switch main && git pull --ff-only origin main

# Rebase your feature branch onto the latest dev
git switch fix/123-audio-init-stall
git fetch origin
git rebase origin/dev
git push --force-with-lease

# Show what will promote from dev → beta
git log --oneline origin/beta..origin/dev
```

**Optional aliases (add to \~/.gitconfig):**

```ini
[alias]
  up = !"git fetch origin && git pull --ff-only"
  graph = log --oneline --graph --decorate --all
  promote-dev-beta = !"git fetch origin && git switch beta && git up origin beta && git merge --no-ff origin/dev && git push origin beta"
  promote-beta-main = !"git fetch origin && git switch main && git up origin main && git merge --no-ff origin/beta && git push origin main"
```

---

### 13) CI gates (suggested)

* **dev PRs:** build + unit tests + lint
* **beta promotion:** full integration tests + smoke on target devices
* **main promotion:** release build, provenance/signing, artifact upload

---

### TL;DR flow (feature/bug)

1. Create issue → define acceptance criteria.
2. `git switch dev && git pull` → `git switch -c type/123-title`
3. Work in small commits with good messages.
4. Rebase your branch on `origin/dev` as needed.
5. Draft PR → `dev`, link issue (`Fixes #123`).
6. Pass checks → Squash Merge into `dev`.
7. Promote `dev → beta` when ready; test with users.
8. Promote `beta → main`, tag release.
9. Hotfixes start on `main`, then back-merge to `beta` and `dev`.
