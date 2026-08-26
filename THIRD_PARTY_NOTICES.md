# Third-party notices

Toudocu is licensed under Apache-2.0. The components below are separately
licensed and retain their own copyright and license notices. Release packaging
generates one complete `THIRD_PARTY_NOTICES.md` by combining this inventory with
the notices of the embedded browser assets.

## Frontend development toolchain

TypeScript, esbuild and Playwright are pinned development dependencies used to
type-check, build and test `web/`. They are not embedded as runtime packages and
are not required by the released Go binary. Their license notices remain in the
npm dependency distributions and lockfile; only the deterministic build output
and separately listed vendored browser libraries are shipped.

## CodeMirror 6

The serve-only editor bundles `codemirror@6.0.2`,
`@codemirror/lang-markdown@6.5.1`, `@codemirror/lang-json@6.0.2`,
`@codemirror/lang-yaml@6.1.3`, `@codemirror/lang-go@6.0.1`,
`@codemirror/lang-java@6.0.2`, `@codemirror/lang-javascript@6.2.5`,
`@codemirror/lint@6.9.7`,
`@codemirror/merge@6.12.2` and their CodeMirror /
Lezer transitive packages under the MIT License. The complete license notice is
embedded as `assets/codemirror.LICENSE.txt`; release checksums are embedded as
`assets/codemirror.checksums.txt`. Node.js is used only to rebuild the vendored
IIFE bundle and is not a Go or end-user runtime dependency.

## go.yaml.in/yaml/v3

- Package: `go.yaml.in/yaml/v3`
- Version: `v3.0.5`
- Source: <https://go.yaml.in/yaml/v3>
- License: Apache-2.0 and MIT

The package parses OpenAPI YAML for deterministic semantic comparison. It is
linked into the Toudocu binary and does not execute input or require a runtime
service. Its source distribution contains MIT-licensed libyaml-derived files
and Apache-2.0-licensed remaining files; its copyright notices are retained in
the module source and this attribution accompanies the binary release.

Copyright (c) 2006-2010 Kirill Simonov

Copyright (c) 2006-2011 Kirill Simonov

Copyright (c) 2011-2019 Canonical Ltd

The libyaml-derived files use the following MIT License; the Apache-2.0 terms
for the remaining files are provided by the top-level `LICENSE` included in
every release archive.

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

## Goldmark

- Package: `github.com/yuin/goldmark`
- Version: `v1.8.5`
- Source: <https://github.com/yuin/goldmark>
- License: MIT

Goldmark parses CommonMark and the enabled GFM extensions into the internal
Markdown AST. It is linked into the Toudocu binary and requires no runtime
service, network access or CGO.

Copyright (c) 2019 Yusuke Inuzuka

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

## Go PTY

- Package: `github.com/aymanbagabas/go-pty`
- Version: `v0.2.2`
- Source: <https://github.com/aymanbagabas/go-pty>
- License: MIT

Terminal Mode links this package into the Toudocu binary for Unix PTY and
Windows ConPTY support. Its transitive runtime packages are
`github.com/creack/pty@v1.1.21` (MIT),
`github.com/u-root/u-root@v0.11.0` (BSD-3-Clause),
`golang.org/x/crypto@v0.33.0` (BSD-3-Clause), and
`golang.org/x/sys@v0.30.0` (BSD-3-Clause). They require no runtime service or
CGO. Copyright (c) 2023 Ayman Bagabas; Copyright (c) 2011 Keith Rarick;
Copyright (c) 2012-2019 u-root Authors; Copyright (c) 2009 The Go Authors.

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

The BSD-3-Clause components permit redistribution and use in source and binary
forms, with or without modification, provided that source redistributions
retain the copyright notice, conditions and disclaimer; binary
redistributions reproduce them in documentation or other materials; and the
copyright holders' or contributors' names are not used to endorse derived
products without prior written permission. These components are provided
"AS IS", without express or implied warranties; their authors are not liable
for direct, indirect, incidental, special, exemplary or consequential damages.

## Xterm.js

Terminal Mode embeds `@xterm/xterm@6.0.0` and
`@xterm/addon-fit@0.11.0`, both under the MIT License. Their complete upstream
license text and copyright notices are generated into
`assets/licenses.json` and included in every release notice bundle. They are
serve-only dynamic assets and are absent from the static runtime.

## Swagger UI

- Package: `swagger-ui-dist`
- Version: `5.32.12`
- Source: <https://github.com/swagger-api/swagger-ui>
- License: Apache-2.0

The canonical serve-only API documentation embeds `swagger-ui.css`,
`swagger-ui-bundle.js` and `swagger-ui-standalone-preset.js`. The upstream
license and generated bundle notices are embedded beside them as
`assets/swagger-ui*.LICENSE.txt`; `assets/swagger-ui.checksums.txt` records
their SHA-256 digests. Node.js is used only to refresh these vendored files and
is not a Go or end-user runtime dependency.

## Mermaid Tiny and bundled dependencies

- Package: `@mermaid-js/tiny`
- Version: `11.16.0`
- Source: <https://www.npmjs.com/package/@mermaid-js/tiny/v/11.16.0>
- License: MIT
- Embedded file: `assets/mermaid.tiny.js`
- SHA-256: `a1bc2282b3d935693780f77931382c517e72eb72ff3427752cbb29941de11bee`

`mermaid.tiny.js` also includes the following third-party code:

- lodash — MIT;
- js-yaml — MIT;
- DOMPurify — Apache-2.0 option (the package is dual-licensed under
  Apache-2.0 or MPL-2.0).

The complete notices and license texts for this browser bundle are stored in
`assets/mermaid.LICENSE.txt` and copied into every generated portal next to the
embedded library. Release packaging also includes this notice as
part of the generated `THIRD_PARTY_NOTICES.md`.
