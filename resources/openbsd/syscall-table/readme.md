# OpenBSD syscall table

Searchable table of OpenBSD system calls, generated from
`sys/kern/syscalls.master` of the **OpenBSD 8.0** release source tarball
(<https://cdn.openbsd.org/pub/OpenBSD/8.0/sys.tar.gz>).

## Files

- `syscalls.master` — the vendored `sys/kern/syscalls.master` from the
  OpenBSD 8.0 `sys.tar.gz` (see the `$OpenBSD: ...,v 1.273 ...` RCS id in its
  header for the exact revision).
- `generate.go` — parses `syscalls.master` and emits `index.html` (from
  `index.html.tmpl`) and `syscalls.csv`.
- `index.html` / `syscalls.csv` — generated output, committed so the static
  site can serve it.

## Regenerate

```sh
go run generate.go
```

## Updating to a new release

```sh
curl -s https://cdn.openbsd.org/pub/OpenBSD/<version>/sys.tar.gz |
    tar -xzO sys/kern/syscalls.master > syscalls.master
# bump Version in generate.go, then:
go run generate.go
```

Manual page links use `https://man.openbsd.org/OpenBSD-<version>/<syscall>.2`.
