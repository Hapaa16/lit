package utils

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Hapaa16/lit/config"
)

type IBlob interface {
	FileToBlob(fileName string) (string, string, error)
}

type BlobFile struct {
	Mode string `json:"mode"`
	Hash string `json:"hash"`
	Path string `json:"path"`
}

type IndexJson map[string]BlobFile

func FileToBlob(fileName string) (string, string, string, error) {

	f, err := os.Open(fileName)

	if err != nil {
		return "", "", "", err
	}

	defer f.Close()

	info, err := f.Stat()

	if err != nil {
		return "", "", "", err
	}

	blobmsg := fmt.Sprintf("blob %d\000", info.Size())
	mode := info.Mode()

	gitMode := GetGitMode(mode)

	hash := sha1.New()

	_, err = hash.Write([]byte(blobmsg))

	if err != nil {
		return "", "", "", err
	}

	if _, err := io.Copy(hash, f); err != nil {
		return "", "", "", err
	}

	h := hex.EncodeToString(hash.Sum(nil))

	return h, blobmsg, gitMode, nil
}

func (index IndexJson) CreateTree() (string, error) {

	return buildTree("", index)
}

func CreateCommitObjectWithTree(treeSha string, commitMsg string, parentHashes []string) error {
	var commitBuffer bytes.Buffer

	hostname, err := os.Hostname()

	if err != nil {
		return err
	}
	// timestamp := "1742885579 +0800"
	author := fmt.Sprintf("author %s <%s> %d +0000\n", hostname, hostname, time.Now().Unix())

	commitBuffer.WriteString(fmt.Sprintf("tree %s\n", treeSha))

	if len(parentHashes) > 0 {
		for _, parentHash := range parentHashes {
			commitBuffer.WriteString(fmt.Sprintf("parent %s\n", parentHash))
		}
	}

	commitBuffer.WriteString(fmt.Sprintf("author %s", author))
	commitBuffer.WriteString(fmt.Sprintf("committer %s", author))
	commitBuffer.WriteString("\n")
	commitBuffer.WriteString(commitMsg + "\n")

	rawCommit := commitBuffer.Bytes()

	header := fmt.Sprintf("commit %d\u0000", len(rawCommit))

	fullCommit := append([]byte(header), rawCommit...)

	commitSha := sha1.Sum(fullCommit)

	commitShaHex := hex.EncodeToString(commitSha[:])

	var compressed bytes.Buffer
	w := zlib.NewWriter(&compressed)
	w.Write(fullCommit)
	w.Close()

	cwd, _ := os.Getwd()

	repoRoot := FindRepoRoot(cwd)

	if repoRoot == "" {
		return fmt.Errorf("could not find .lit repo root")
	}

	objectDir := path.Join(repoRoot, ".lit", "objects", commitShaHex[:2])
	objectPath := path.Join(objectDir, commitShaHex[2:])

	os.MkdirAll(objectDir, 0755)
	os.WriteFile(objectPath, compressed.Bytes(), 0644)

	head := GetCurrentBranch()

	err = HandleHeadFile(head, commitShaHex)

	return err
}

func HandleHeadFile(HEAD string, commitHash string) error {

	path := path.Join(config.InitDirName, string(os.PathSeparator), HEAD)

	err := ioutil.WriteFile(path, []byte(commitHash), 0644)

	if err != nil {
		return fmt.Errorf("error writing file: %v", err)
	}

	return nil
}

func GetCurrentBranch() string {

	wd, err := os.Getwd()

	if err != nil {
		fmt.Println("Lit error: ", err)
		os.Exit(1)
	}

	rootRepo := FindRepoRoot(wd)

	headFile, err := os.ReadFile(rootRepo + "/.lit/HEAD")

	if err != nil {
		fmt.Println("Lit error: ", err)
		os.Exit(1)
	}

	headContent := string(headFile)

	splittedHead := strings.Split(headContent, ": ")[1]

	return splittedHead
}

func GetGitMode(mode fs.FileMode) string {
	var gitMode string

	switch {

	case mode&os.ModeSymlink != 0:
		gitMode = "120000"
	case mode.IsDir():
		gitMode = "040000"
	case mode&0111 != 0:
		gitMode = "100755" // executable bit is set
	default:
		gitMode = "100644" // regular file
	}

	return gitMode
}

func FindRepoRoot(start string) string {
	for {
		if _, err := os.Stat(filepath.Join(start, ".lit")); err == nil {
			return start
		}

		parent := filepath.Dir(start)

		if parent == start {
			return ""
		}
		start = parent
	}
}

func buildTree(currentDir string, index IndexJson) (string, error) {
	var treeBuf bytes.Buffer

	children := make(map[string]IndexJson)

	var entries []struct {
		Mode string
		Name string
		Hash string
	}

	for filePath, entry := range index {
		relPath := filePath

		if currentDir != "" {
			if !strings.HasPrefix(filePath, currentDir+"/") {
				continue
			}
			relPath = strings.TrimPrefix(filePath, currentDir+"/")

		}

		parts := strings.SplitN(relPath, "/", 2)

		if len(parts) == 1 {
			entries = append(entries, struct {
				Mode string
				Name string
				Hash string
			}{
				Mode: entry.Mode,
				Name: parts[0],
				Hash: entry.Hash,
			})
		} else {
			subdir := parts[0]
			subdirPath := path.Join(currentDir, subdir)

			if _, ok := children[subdirPath]; !ok {
				children[subdirPath] = IndexJson{}
			}
			children[subdirPath][filePath] = entry
		}

	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})

	for _, entry := range entries {
		treeBuf.WriteString(fmt.Sprintf("%s %s", entry.Mode, entry.Name))
		treeBuf.WriteByte(0)

		shaBytes, err := hex.DecodeString(entry.Hash)
		if err != nil {
			return "", err
		}
		treeBuf.Write(shaBytes)
	}

	for subdirPath, subIndex := range children {

		subtreeSha, err := buildTree(subdirPath, subIndex)

		if err != nil {
			return "", err
		}

		subdirName := path.Base(subdirPath)

		treeBuf.WriteString(fmt.Sprintf("040000 %s", subdirName))
		treeBuf.WriteByte(0)

		shaBytes, err := hex.DecodeString(subtreeSha)
		if err != nil {
			return "", err
		}
		treeBuf.Write(shaBytes)
	}

	rawTree := treeBuf.Bytes()

	header := fmt.Sprintf("tree %d\u0000", len(rawTree))

	fullTree := append([]byte(header), rawTree...)
	treeSha := sha1.Sum(fullTree)
	treeShaHex := hex.EncodeToString(treeSha[:])

	var compressed bytes.Buffer
	w := zlib.NewWriter(&compressed)
	w.Write(fullTree)
	w.Close()

	cwd, err := os.Getwd()

	if err != nil {
		return "", err
	}

	rootPath := FindRepoRoot(cwd)

	objectDir := path.Join(rootPath, ".lit/objects", treeShaHex[:2])
	objectPath := path.Join(objectDir, treeShaHex[2:])

	os.MkdirAll(objectDir, 0755)
	os.WriteFile(objectPath, compressed.Bytes(), 0644)

	return treeShaHex, nil

}

func GetLatestCommit() (string, error) {
	head := GetCurrentBranch()

	wd, err := os.Getwd()

	if err != nil {
		return "", err
	}

	rootRepo := FindRepoRoot(wd)

	commitPath := path.Join(rootRepo, string(os.PathSeparator)+".lit", head)

	fmt.Println(commitPath)

	file, err := os.ReadFile(commitPath)

	if err != nil {
		return "", err
	}

	return string(file), nil
}
