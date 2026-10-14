---
title: canopy
description: A small, read-only web viewer for git repositories, in the spirit of cgit. One Go binary, no JavaScript. Push over SSH; everyone else browses and clones over HTTPS.
---

## What it does

**Browse**: An index of every repository, grouped by folder. Summaries
with the README, trees with syntax highlighting, rendered markdown, raw
files, blame, history, commits with diffs and `.patch` downloads, refs
and tarballs.

**Clone**: Anything public clones over HTTPS with git's smart HTTP
protocol. Pushing over HTTP is never possible: canopy has no write path
at all.

**Push**: Over SSH to a locked-down `git` account that runs nothing but
git. Pushing to a name that doesn't exist yet creates the repository.

**Deploy**: One command from your own machine sets up canopy, HTTPS, bot
protection, pushing and a hardened server, optionally with SSH on a
private [Tailscale](https://tailscale.com) network.

**Feeds and activity**: Atom feeds of commits and tags for every
repository, and an optional activity tab with a year of commits as a
GitHub-style graph.

**Themes**: Light and dark colour themes (GitHub, Ayu, Night Owl,
Catppuccin) or your own, and Google Fonts that canopy downloads and
serves itself, so visitors never contact Google.

**Private repositories**: `ssh gitserver private NAME on` takes a
repository off the web site within 15 seconds. Over SSH you push and
fetch as usual.

**Mirrors**: Follow repositories hosted elsewhere under `[mirrors]`.
Visitors browse them here and are pointed to the upstream to clone.

**Signatures**: Commits and tags signed with an SSH key listed under
`[signatures]` show as verified. canopy checks the signatures itself and
never runs gpg or `ssh-keygen`.

[Read the docs](/docs/), [see it running](https://git.obk.dev/), or
[clone it](/clone/).
