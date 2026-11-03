# web — Frontend Project Collection

This directory contains the business area's frontend projects: the admin console application and the dynamic form SDK. Both use Vue 3 + TypeScript + Vite.

## Project List

### [`admin/`](./admin) — Admin Console

Admin console single-page application built with Vue 3 + TypeScript + Vite + Naive UI + Pinia + Vue Router, using development port `5173`.

- Layout: a full-height white menu on the left and a three-section layout of white floating cards on a light gray-green background on the right; the theme color is green `#18a058`.
- The phone bar is absolutely centered in the Header and remains available globally; switching menus does not affect the call state.
- The phone bar's source code is not part of this directory. The core private repository's `phonebar/` provides its obfuscated build artifacts, which are synchronized to `admin/public/phonebar/`.

Common commands:

```bash
cd business/web/admin
npm run dev          # Local development
npm run type-check   # Type checking (vue-tsc)
npm run build        # Type checking + production build
```

### [`form/`](./form) — Dynamic Form SDK

Designer, renderer, and instance management for dynamic forms, built as an embeddable library. Built with Vue 3 + TypeScript + Vite + Naive UI + vue-draggable-plus.

- Vite library mode, IIFE format, with the global name `VocaraForm`.
- Build artifacts `vocara-form.js` / `vocara-form.css` are written directly to `admin/public/form/` for the admin console to load.

Common commands:

```bash
cd business/web/form
npm run build        # Type checking + library build, output to ../admin/public/form
npm run test:e2e     # Smoke checks
```

## Build Order

Both the form SDK and phone bar artifacts are placed under `admin/public/`. In a fresh environment, build `form` first (and, when needed, build the phone bar in the core repository with `make build-phonebar`), then build `admin` so that the admin console packages the latest component artifacts.
