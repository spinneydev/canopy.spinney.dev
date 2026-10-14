---
title: Clone
description: Clone canopy, try it with sample repositories, and serve your own.
---

<p class="lead">Get the source, run it with sample repositories in a
minute, then point it at your own.</p>

## 1. Get the source

```sh
git clone https://github.com/spinneydev/canopy
```

## 2. Try the demo

It needs only Go and git, and works on Linux, macOS and Windows.

```sh
cd canopy
# sample repositories in demo/repos
go run ./demo
# http://127.0.0.1:8080
go run . -config demo/canopy.toml
```

The demo has a C project with releases, merges and branches, a
`.dotfiles` repository with symlinks, markdown notes with images, a
nested repository, an empty one, and one hidden by the config.

## 3. Serve your own repositories

```sh
go build
./canopy -root ~/src -listen 127.0.0.1:8080
```

Every repository under `-root` shows up in the index, grouped by folder.

## 4. Put it on a server

One command from your machine sets up canopy, HTTPS, bot protection,
pushing over SSH and a hardened server:

```sh
./install.sh root@git.example.com
```

See [deploying](/docs/deploy/) for the whole walk-through, from a fresh
VPS to backups.

<dl class="facts">
<dt>Language</dt><dd>Go, plus the git binary</dd>
<dt>Runs on</dt><dd>Debian 13, Ubuntu 24.04 and 26.04, Fedora 44</dd>
<dt>Develop on</dt><dd>Linux, macOS, Windows</dd>
<dt>Licence</dt><dd>GPL-3.0 or later</dd>
</dl>
