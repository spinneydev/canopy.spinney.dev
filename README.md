# canopy.spinney.dev

The site of canopy, the read-only web viewer for git: a home page, a
clone page, and canopy's documentation, in the look of spinney.dev.

The pages are markdown, and a small Go program sets them in templates:

    content/            the site's own pages: home and clone
    templates/          layout.html around those, docs.html around the docs
    main.go             the build: content/*.md and canopy's markdown
                        into public/
    public/site.css     the look
    public/             the site; index.html, clone/, docs/ and chroma.css
                        are built

The documentation is not kept here: it is canopy's own markdown, such as
`deploy.md` and `usage.md`, read from a checkout of canopy next to this
folder, `../canopy`, or wherever `-src` points; with nothing there, `main.go` clones canopy from GitHub into it
first. The list of documents and their order is `docs` in `main.go`.

A page in `content/` starts with its title, and optionally a description,
between two `---` lines:

    ---
    title: Clone
    description: Clone canopy, try it with sample repositories, and serve your own.
    ---

    ## 1. Get the source

`content/home.md` is the home page; any other `content/<name>.md` is
served at `/<name>/`.

To try it:

    go run . -serve     # builds, then serves on 127.0.0.1:8000 and
                        # rebuilds on every page load

To build it on Cloudflare Pages, which fetches canopy by itself:

    build command:      go run .
    output directory:   public

The built pages are not committed, so the documentation is as new as the
last build.

## License

[GNU GPL v3](LICENSE), or any later version.

    canopy.spinney.dev, canopy's site and the program that builds it.
    Copyright (C) 2026 The spinney authors

    This program is free software: you can redistribute it and/or modify it
    under the terms of the GNU General Public License as published by the
    Free Software Foundation, either version 3 of the License, or (at your
    option) any later version.

    This program is distributed in the hope that it will be useful, but
    WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU General
    Public License for more details.
