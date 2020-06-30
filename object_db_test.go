package gitobj

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const roundTripCommitSha string = `561ed224a6bd39232d902ad8023c0ebe44fbf6c5`
const roundTripCommit string = `tree f2ebdf9c967f69d57b370901f9344596ec47e51c
parent fe8fbf7de1cd9f08ae642e502bf5de94e523cc08
author brian m. carlson <bk2204@github.com> 1543506816 +0000
committer brian m. carlson <bk2204@github.com> 1543506816 +0000
gpgsig -----BEGIN PGP SIGNATURE-----
 Version: GnuPG/MacGPG2 v2.2.9 (Darwin)
 
 iQIGBAABCgAwFiEETbktHYzuflTwZxNFLQybwS+Cs6EFAlwAC4cSHGJrMjIwNEBn
 aXRodWIuY29tAAoJEC0Mm8EvgrOhiRMN/2rTxkBb5BeQQeq7rPiIW8+29FzuvPeD
 /DhxlRKwKut9h4qhtxNQszTezxhP4PLOkuMvUax2pGXCQ8cjkSswagmycev+AB4d
 s0loG4SrEwvH8nAdr6qfNx4ZproRJ8QaEJqyN9SqF7PCWrUAoJKehdgA38WtYFws
 ON+nIwzDIvgpoNI+DzgWrx16SOTp87xt8RaJOVK9JNZQk8zBh7rR2viS9CWLysmz
 wOh3j4XI1TZ5IFJfpCxZzUDFgb6K3wpAX6Vux5F1f3cN5MsJn6WUJCmYCvwofeeZ
 6LMqKgry7EA12l7Tv/JtmMeh+rbT5WLdMIsjascUaHRhpJDNqqHCKMEj1zh3QZNY
 Hycdcs24JouVAtPwg07f1ncPU3aE624LnNRA9A6Ih6SkkKE4tgMVA5qkObDfwzLE
 lWyBj2QKySaIdSlU2EcoH3UK33v/ofrRr3+bUkDgxdqeV/RkBVvfpeMwFVSFWseE
 bCcotryLCZF7vBQU+pKC+EaZxQV9L5+McGzcDYxUmqrhwtR+azRBYFOw+lOT4sYD
 FxdLFWCtmDhKPX5Ajci2gmyfgCwdIeDhSuOf2iQQGRpE6y7aka4AlaE=
 =UyqL
 -----END PGP SIGNATURE-----

pack/set: ignore packs without indices

When we look for packs to read, we look for a pack file, and then an
index, and fail if either one is missing.  When Git looks for packs to
read, it looks only for indices and then checks if the pack is present.

The Git approach handles the case when there is an extra pack that lacks
an index, while our approach does not.  Consequently, we can get various
errors (showing up so far only on Windows) when an index is missing.

If the index file cannot be read for any reason, simply skip the entire
pack altogether and continue on.  This leaves us no more or less
functional than Git in terms of discovering objects and makes our error
handling more robust.
`

func TestDecodeObject(t *testing.T) {
	sha := "af5626b4a114abcb82d63db7c8082c3c4756e51b"
	contents := "Hello, world!\n"

	var buf bytes.Buffer

	zw := zlib.NewWriter(&buf)
	fmt.Fprintf(zw, "blob 14\x00%s", contents)
	zw.Close()

	b, err := NewMemoryBackend(map[string]io.ReadWriter{
		sha: &buf,
	})
	require.NoError(t, err)

	odb, err := FromBackend(b)
	require.NoError(t, err)

	shaHex, _ := hex.DecodeString(sha)
	obj, err := odb.Object(shaHex)
	blob, ok := obj.(*Blob)

	require.NoError(t, err)
	require.True(t, ok)

	got, err := ioutil.ReadAll(blob.Contents)
	assert.Nil(t, err)
	assert.Equal(t, contents, string(got))
}

func TestDecodeBlob(t *testing.T) {
	sha := "af5626b4a114abcb82d63db7c8082c3c4756e51b"
	contents := "Hello, world!\n"

	var buf bytes.Buffer

	zw := zlib.NewWriter(&buf)
	fmt.Fprintf(zw, "blob 14\x00%s", contents)
	zw.Close()

	b, err := NewMemoryBackend(map[string]io.ReadWriter{
		sha: &buf,
	})
	require.NoError(t, err)

	odb, err := FromBackend(b)
	require.NoError(t, err)

	shaHex, _ := hex.DecodeString(sha)
	blob, err := odb.Blob(shaHex)

	assert.Nil(t, err)
	assert.EqualValues(t, 14, blob.Size)

	got, err := ioutil.ReadAll(blob.Contents)
	assert.Nil(t, err)
	assert.Equal(t, contents, string(got))
}

func TestDecodeTree(t *testing.T) {
	sha := "fcb545d5746547a597811b7441ed8eba307be1ff"
	hexSha, err := hex.DecodeString(sha)
	require.Nil(t, err)

	blobSha := "e69de29bb2d1d6434b8b29ae775ad8c2e48c5391"
	hexBlobSha, err := hex.DecodeString(blobSha)
	require.Nil(t, err)

	var buf bytes.Buffer

	zw := zlib.NewWriter(&buf)
	fmt.Fprintf(zw, "tree 37\x00")
	fmt.Fprintf(zw, "100644 hello.txt\x00")
	zw.Write(hexBlobSha)
	zw.Close()

	b, err := NewMemoryBackend(map[string]io.ReadWriter{
		sha: &buf,
	})
	require.NoError(t, err)

	odb, err := FromBackend(b)
	require.NoError(t, err)

	tree, err := odb.Tree(hexSha)

	assert.Nil(t, err)
	require.Equal(t, 1, len(tree.Entries))
	assert.Equal(t, &TreeEntry{
		Name:     "hello.txt",
		Oid:      hexBlobSha,
		Filemode: 0100644,
	}, tree.Entries[0])
}

func TestDecodeCommit(t *testing.T) {
	sha := "d7283480bb6dc90be621252e1001a93871dcf511"
	commitShaHex, err := hex.DecodeString(sha)
	assert.Nil(t, err)

	var buf bytes.Buffer

	zw := zlib.NewWriter(&buf)
	fmt.Fprintf(zw, "commit 173\x00")
	fmt.Fprintf(zw, "tree fcb545d5746547a597811b7441ed8eba307be1ff\n")
	fmt.Fprintf(zw, "author Taylor Blau <me@ttaylorr.com> 1494620424 -0600\n")
	fmt.Fprintf(zw, "committer Taylor Blau <me@ttaylorr.com> 1494620424 -0600\n")
	fmt.Fprintf(zw, "\ninitial commit\n")
	zw.Close()

	b, err := NewMemoryBackend(map[string]io.ReadWriter{
		sha: &buf,
	})
	require.NoError(t, err)

	odb, err := FromBackend(b)
	require.NoError(t, err)

	commit, err := odb.Commit(commitShaHex)

	assert.Nil(t, err)
	assert.Equal(t, "Taylor Blau <me@ttaylorr.com> 1494620424 -0600", commit.Author)
	assert.Equal(t, "Taylor Blau <me@ttaylorr.com> 1494620424 -0600", commit.Committer)
	assert.Equal(t, "initial commit", commit.Message)
	assert.Equal(t, 0, len(commit.ParentIDs))
	assert.Equal(t, "fcb545d5746547a597811b7441ed8eba307be1ff", hex.EncodeToString(commit.TreeID))
}

func TestWriteBlob(t *testing.T) {
	b, err := NewMemoryBackend(nil)
	require.NoError(t, err)

	odb, err := FromBackend(b)
	require.NoError(t, err)

	sha, err := odb.WriteBlob(&Blob{
		Size:     14,
		Contents: strings.NewReader("Hello, world!\n"),
	})

	expected := "af5626b4a114abcb82d63db7c8082c3c4756e51b"

	_, s := b.Storage()

	assert.Nil(t, err)
	assert.Equal(t, expected, hex.EncodeToString(sha))
	assert.NotNil(t, s.(*memoryStorer).fs[hex.EncodeToString(sha)])
}

func TestWriteTree(t *testing.T) {
	b, err := NewMemoryBackend(nil)
	require.NoError(t, err)

	odb, err := FromBackend(b)
	require.NoError(t, err)

	blobSha := "e69de29bb2d1d6434b8b29ae775ad8c2e48c5391"
	hexBlobSha, err := hex.DecodeString(blobSha)
	require.Nil(t, err)

	sha, err := odb.WriteTree(&Tree{Entries: []*TreeEntry{
		{
			Name:     "hello.txt",
			Oid:      hexBlobSha,
			Filemode: 0100644,
		},
	}})

	expected := "fcb545d5746547a597811b7441ed8eba307be1ff"

	_, s := b.Storage()

	assert.Nil(t, err)
	assert.Equal(t, expected, hex.EncodeToString(sha))
	assert.NotNil(t, s.(*memoryStorer).fs[hex.EncodeToString(sha)])
}

func TestWriteCommit(t *testing.T) {
	b, err := NewMemoryBackend(nil)
	require.NoError(t, err)

	odb, err := FromBackend(b)
	require.NoError(t, err)

	when := time.Unix(1257894000, 0).UTC()
	author := &Signature{Name: "John Doe", Email: "john@example.com", When: when}
	committer := &Signature{Name: "Jane Doe", Email: "jane@example.com", When: when}

	tree := "fcb545d5746547a597811b7441ed8eba307be1ff"
	treeHex, err := hex.DecodeString(tree)
	assert.Nil(t, err)

	sha, err := odb.WriteCommit(&Commit{
		Author:    author.String(),
		Committer: committer.String(),
		TreeID:    treeHex,
		Message:   "initial commit",
	})

	expected := "fee8a35c2890cd6e0e28d24cc457fcecbd460962"

	_, s := b.Storage()

	assert.Nil(t, err)
	assert.Equal(t, expected, hex.EncodeToString(sha))
	assert.NotNil(t, s.(*memoryStorer).fs[hex.EncodeToString(sha)])
}

func TestWriteCommitWithGPGSignature(t *testing.T) {
	b, err := NewMemoryBackend(nil)
	require.NoError(t, err)

	odb, err := FromBackend(b)
	require.NoError(t, err)

	commit := new(Commit)
	_, err = commit.Decode(
		sha1.New(),
		strings.NewReader(roundTripCommit), int64(len(roundTripCommit)))
	require.NoError(t, err)

	buf := new(bytes.Buffer)
	commit.Encode(buf)
	assert.Equal(t, roundTripCommit, buf.String())

	sha, err := odb.WriteCommit(commit)

	assert.Nil(t, err)
	assert.Equal(t, roundTripCommitSha, hex.EncodeToString(sha))
}

func TestDecodeTag(t *testing.T) {
	const sha = "7639ba293cd2c457070e8446ecdea56682af0f48"
	tagShaHex, err := hex.DecodeString(sha)

	var buf bytes.Buffer

	zw := zlib.NewWriter(&buf)
	fmt.Fprintf(zw, "tag 165\x00")
	fmt.Fprintf(zw, "object 6161616161616161616161616161616161616161\n")
	fmt.Fprintf(zw, "type commit\n")
	fmt.Fprintf(zw, "tag v2.4.0\n")
	fmt.Fprintf(zw, "tagger A U Thor <author@example.com>\n")
	fmt.Fprintf(zw, "\n")
	fmt.Fprintf(zw, "The quick brown fox jumps over the lazy dog.\n")
	zw.Close()

	b, err := NewMemoryBackend(map[string]io.ReadWriter{
		sha: &buf,
	})
	require.NoError(t, err)

	odb, err := FromBackend(b)
	require.NoError(t, err)

	tag, err := odb.Tag(tagShaHex)

	assert.Nil(t, err)

	assert.Equal(t, []byte("aaaaaaaaaaaaaaaaaaaa"), tag.Object)
	assert.Equal(t, CommitObjectType, tag.ObjectType)
	assert.Equal(t, "v2.4.0", tag.Name)
	assert.Equal(t, "A U Thor <author@example.com>", tag.Tagger)
	assert.Equal(t, "The quick brown fox jumps over the lazy dog.", tag.Message)
}

func TestWriteTag(t *testing.T) {
	b, err := NewMemoryBackend(nil)
	require.NoError(t, err)

	odb, err := FromBackend(b)
	require.NoError(t, err)

	sha, err := odb.WriteTag(&Tag{
		Object:     []byte("aaaaaaaaaaaaaaaaaaaa"),
		ObjectType: CommitObjectType,
		Name:       "v2.4.0",
		Tagger:     "A U Thor <author@example.com>",

		Message: "The quick brown fox jumps over the lazy dog.",
	})

	expected := "e614dda21829f4176d3db27fe62fb4aee2e2475d"

	_, s := b.Storage()

	assert.Nil(t, err)
	assert.Equal(t, expected, hex.EncodeToString(sha))
	assert.NotNil(t, s.(*memoryStorer).fs[hex.EncodeToString(sha)])
}

func TestReadingAMissingObjectAfterClose(t *testing.T) {
	sha, _ := hex.DecodeString("af5626b4a114abcb82d63db7c8082c3c4756e51b")

	b, err := NewMemoryBackend(nil)
	require.NoError(t, err)

	ro, rw := b.Storage()

	db := &ObjectDatabase{
		ro:     ro,
		rw:     rw,
		closed: 1,
	}

	blob, err := db.Blob(sha)
	assert.EqualError(t, err, "gitobj: cannot use closed *pack.Set")
	assert.Nil(t, blob)
}

func TestClosingAnObjectDatabaseMoreThanOnce(t *testing.T) {
	db, err := FromFilesystem("/tmp", "")
	assert.Nil(t, err)

	assert.Nil(t, db.Close())
	assert.EqualError(t, db.Close(), "gitobj: *ObjectDatabase already closed")
}

func TestObjectDatabaseRootWithRoot(t *testing.T) {
	db, err := FromFilesystem("/foo/bar/baz", "")
	assert.Nil(t, err)

	root, ok := db.Root()
	assert.Equal(t, "/foo/bar/baz", root)
	assert.True(t, ok)
}

func TestObjectDatabaseRootWithoutRoot(t *testing.T) {
	root, ok := new(ObjectDatabase).Root()

	assert.Equal(t, "", root)
	assert.False(t, ok)
}
