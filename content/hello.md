---
title: "My First Terrific Post with Frontmatter!"
date: "2025-05-26"
author: "Dr. Terrific"
tags: ["go", "ssg", "fun"]
custom_greeting: "Hello from the frontmatter!"
---

This is my very first post using **SHIT SSG** and it now supports frontmatter!

## Markdown Goodness

I can use all the usual Markdown features. The title above and the date should be
pulled from the frontmatter block.

My author is: {{ .Params.author }}
My greeting: {{ .Params.custom_greeting }}