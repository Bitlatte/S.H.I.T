# S.H.I.T (Static HTML Is Terrific!)
 So, you need a static site. You've looked at the big, fancy generators with their gazillion features, webpack configs that require a PhD, and build times longer than your last existential crisis.

Fuck that.

This is SHIT. It's a simple static site generator.
It's small. It's simple. It turns your Markdown into HTML and then politely fucks off. Nothing fancy, just the basics to get your shit online.

# What is this SHIT?
SHIT is a command-line tool written in Go that takes your content (mostly Markdown), shoves it through some templates, and spits out a folder full of static HTML files. You know, for a website.

It's designed for when you just want to write some stuff and have it look vaguely presentable without selling your soul to a JavaScript ecosystem.

# Why this SHIT?
It's Simple: Seriously, the codebase isn't trying to win any awards for complexity.

No Bullshit: No complex dependencies (beyond Go itself for building). No node_modules black hole.

Convention over Configuration (mostly): It expects a certain directory structure, which means less fiddling with config files.

It's Fast Enough: Go is pretty zippy. For most small to medium sites, it'll build faster than you can say "Oh SHIT, I forgot to save."

It's Terrific! (The name says so).

# Features (The Good SHIT)
Markdown Processing: Turns your .md files into .html. Uses Goldmark, so GFM (GitHub Flavored Markdown) is mostly supported.

Frontmatter: Chuck some YAML at the top of your Markdown files for titles, dates, and whatever other crap you want (accessible in templates via .Params.yourKey).

---
title: "My Awesome Page"
date: "2025-12-25"
author: "Some Dude"
---
Your content...

Templating: Uses Go's built-in html/template. Stick your base.html and partials (like header.html, footer.html) in a layouts/ directory.

Include partials like this: {{ template "partials/header.html" . }} (assuming your partial is in layouts/partials/header.html).

Static Asset Handling: Anything you put in a static/ directory gets copied straight to your output folder. CSS, JS, images, cat pictures – go nuts.

Live-Reloading Dev Server: The serve command builds your site, fires up a local web server, and watches for changes in your content/, layouts/, and static/ folders, rebuilding automatically. Just refresh your browser (auto-page-reload is too fancy for this SHIT right now).

Basic Configuration: A simple optional config.yaml for a few settings.

# Getting this SHIT Running (Installation)
Prerequisites:

You need Go installed (like, version 1.20 or newer probably? Whatever's reasonably current).

Option 1: From Source (Recommended for now)

Clone this repository:
```sh
git clone [https://github.com/Bitlatte/S.H.I.T.git](https://github.com/Bitlatte/S.H.I.T.git) ./shit
cd shit
```
Build it:
```sh
go build -o shit .
```
Now you can run ./shit from that directory, or move the shit executable somewhere in your $PATH.

Option 2: go install (If you're feeling fancy)
```sh
go install [github.com/Bitlatte/S.H.I.T@latest](https://github.com/Bitlatte/S.H.I.T@latest)
```
This will install it to your $GOPATH/bin (or $HOME/go/bin). Make sure that's in your $PATH.

# How to Use This SHIT (Usage)
SHIT  expects a certain project layout. It's not super strict, but follow these conventions and your life will be easier.

1. Directory Structure:
```sh
your-shitty-site/
├── content/               # Your Markdown files go here. Subfolders are fine.
│   ├── index.md
│   ├── about.md
│   └── posts/
│       └── my-first-post.md
├── layouts/               # Your HTML templates.
│   ├── base.html          # The main site template.
│   └── partials/          # Optional: for header, footer, etc.
│       ├── header.html
│       └── footer.html
├── static/                # Static files (CSS, JS, images).
│   ├── css/
│   │   └── style.css
│   └── images/
│       └── logo.png
├── config.yaml            # Optional: configuration.
└── public/                # Default output directory (where the generated site goes).
                           # This gets created/cleared by the build command.
```
2. Configuration (config.yaml - Optional):

Create a config.yaml in your project root if you want to override defaults:
```yaml
outputDir: "dist"  # Default is "public"
baseURL: "[https://your-shitty-site.com](https://your-shitty-site.com)" # Default is ""
siteTitle: "My Absolutely Terrific SHIT Site" # Default is "My Terrific SHIT Site"
```
3. Commands:

Build the site:
```sh
shit build
```
This processes everything and puts the static site into your outputDir (default public/).

Serve locally & watch for changes:
```sh
shit serve
```
Or specify a port:
```sh
shit serve -p 8080
```
This builds the site, starts a local web server (default http://localhost:1313), and watches your content/, layouts/, and static/ directories. When you save a file, it rebuilds. You'll need to manually refresh your browser. Press Ctrl+C to stop.

# Data in Templates
Your layouts/base.html (and any partials it calls) will receive a PageData object. You can access its fields like:

{{ .SiteTitle }}: Global site title from config.yaml.

{{ .PageTitle }}: Title of the current page (from frontmatter or filename).

{{ .Content }}: The HTML rendered from your Markdown. (Use template.HTML type in Go struct, so use {{ .Content }} not {{ .Content | safeHTML }} etc.)

{{ .BaseURL }}: Base URL from config.yaml.

{{ .Date }}: Date from frontmatter (as a string).

{{ .Params.your_frontmatter_key }}: Access any other custom frontmatter variables.

Example:

<!DOCTYPE html>
<html>
<head>
    <title>{{ .PageTitle }} | {{ .SiteTitle }}</title>
    <link rel="stylesheet" href="{{ .BaseURL }}/css/style.css">
</head>
<body>
    {{ template "partials/header.html" . }}
    <main>
        {{ if .Params.custom_intro }}
            <p>{{ .Params.custom_intro }}</p>
        {{ end }}
        {{ .Content }}
    </main>
    {{ template "partials/footer.html" . }}
</body>
</html>

# Contributing (Want to add more SHIT?)
Sure, why the hell not? If you have an idea that keeps it simple and doesn't add a metric fuckton of complexity, feel free to open an issue or a pull request.

Just remember the ethos: Static HTML Is Terrific (and simple).

# License (What's the deal with this SHIT?)
This project is licensed under the GLWTPL License. Basically, do whatever you want with it, but don't blame me if your computer explodes or your cat learns to code and rewrites it in Rust. See the LICENSE file for more details.

And that's it. Go make some SHIT. Or don't. I'm not your mom.