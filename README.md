# gitobj

Package `gitobj` reads and writes loose and packed Git objects.

## Getting Started

To access a repository's objects, begin by "opening" that repository for use:

```go
package main

import (
	"github.com/git-lfs/gitobj"
)

func main() {
	repo, err := gitobj.FromFilesystem("/path/to/repo.git", "")
	if err != nil {
		panic(err)
	}
	defer repo.Close()
}
```

You can then open objects for inspection with the `Blob()`, `Commit()`,
`Tag()`, or `Tree()` functions:

```go
func main() {
	repo, err := gitobj.FromFilesystem("/path/to/repo.git", "")
	if err != nil {
		panic(err)
	}
	defer repo.Close()

	commit, err := repo.Commit([]byte{...})
	if err != nil {
		panic(err)
	}
}
```

Once an object is opened or an instance is held, it can be saved to the object
database using the `WriteBlob()`, `WriteCommit()`, `WriteTag()`, or
`WriteTree()` functions:

```go
func main() {
	repo, err := gitobj.FromFilesystem("/path/to/repo.git", "")
	if err != nil {
		panic(err)
	}
	defer repo.Close()

	commit, err := repo.Commit([]byte{...})
	if err != nil {
		panic(err)
	}

	commit.Message = "Hello from gitobj!"
	commit.ExtraHeaders = append(commit.ExtraHeaders, &gitobj.ExtraHeader{
		K: "Signed-off-by",
		V: "Jane Doe <jane@example.com>",
	})

	if _, err := repository.WriteCommit(commit); err != nil {
		panic(err)
	}
}
```

### Packed Objects

Package `gitobj` has support for reading "packed" objects (i.e., objects found
in [packfiles][1]) via package `github.com/git-lfs/gitobj/pack`. Package `pack`
implements searching pack index (`.idx`) files and locating the corresponding
delta-base chain in the appropriate `pack` file. It understands both version
1 and version 2 of the packfile specification.

`gitobj` will always try to locate a loose object first. If a loose object
cannot be found with the appropriate SHA-1, the repository's packfile(s) will
be searched. If an object is located in a packfile, that object will be
reconstructed along its delta-base chain and then returned transparently.

### More information

For more: https://godoc.org/github.com/git-lfs/gitobj.

## License

MIT.

[1]: https://git-scm.com/book/en/v2/Git-Internals-Packfiles
