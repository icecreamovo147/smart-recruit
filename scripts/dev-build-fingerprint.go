// Command dev-build-fingerprint computes a content fingerprint for a Go command
// and its local workspace dependencies. It is used by start-dev.sh to avoid
// relinking unchanged development binaries.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

const fingerprintSchema = "smart-recruit-dev-build-v1"

var toolRevision = "development"

type stringList []string

func (values *stringList) String() string { return strings.Join(*values, ",") }

func (values *stringList) Set(value string) error {
	*values = append(*values, value)
	return nil
}

type module struct {
	Path    string
	Version string
	Sum     string
	GoMod   string
	Replace *module
}

type goPackage struct {
	ImportPath   string
	Dir          string
	GoFiles      []string
	CgoFiles     []string
	CFiles       []string
	CXXFiles     []string
	MFiles       []string
	HFiles       []string
	FFiles       []string
	SFiles       []string
	SwigFiles    []string
	SwigCXXFiles []string
	SysoFiles    []string
	EmbedFiles   []string
	Module       *module
}

func main() {
	var extras stringList
	root := flag.String("root", "", "repository root")
	dir := flag.String("dir", "", "Go command working directory")
	pkg := flag.String("package", "", "Go command package")
	tags := flag.String("tags", "", "Go build tags")
	inputsOnly := flag.Bool("inputs-only", false, "hash only explicitly supplied inputs")
	flag.Var(&extras, "extra", "additional file or directory to include (repeatable)")
	flag.Parse()

	if *root == "" {
		fatalf("--root is required")
	}
	absRoot, err := filepath.Abs(*root)
	if err != nil {
		fatalf("resolve repository root: %v", err)
	}
	if resolvedRoot, resolveErr := filepath.EvalSymlinks(absRoot); resolveErr == nil {
		absRoot = resolvedRoot
	}

	h := sha256.New()
	writeValue(h, "schema", fingerprintSchema)
	writeValue(h, "tool-revision", toolRevision)
	writeValue(h, "runtime", runtime.Version())
	files := make(map[string]struct{})

	if !*inputsOnly {
		if *dir == "" || *pkg == "" {
			fatalf("--dir and --package are required unless --inputs-only is used")
		}
		writeValue(h, "package", *pkg)
		writeValue(h, "tags", *tags)
		collectGoInputs(h, files, absRoot, *dir, *pkg, *tags)
		collectGoEnvironment(h, *dir)
		addIfPresent(files, filepath.Join(absRoot, "go.work"))
		addIfPresent(files, filepath.Join(absRoot, "go.work.sum"))
	}

	for _, extra := range extras {
		collectExtra(files, extra)
	}
	hashFiles(h, files, absRoot)
	fmt.Println(hex.EncodeToString(h.Sum(nil)))
}

func collectGoInputs(h hash.Hash, files map[string]struct{}, root, dir, pkg, tags string) {
	args := []string{"list", "-deps", "-json"}
	if tags != "" {
		args = append(args, "-tags", tags)
	}
	args = append(args, pkg)
	command := exec.Command("go", args...)
	command.Dir = dir
	stdout, err := command.StdoutPipe()
	if err != nil {
		fatalf("read go list output: %v", err)
	}
	command.Stderr = os.Stderr
	if err := command.Start(); err != nil {
		fatalf("start go list: %v", err)
	}

	decoder := json.NewDecoder(stdout)
	for {
		var pkgInfo goPackage
		if err := decoder.Decode(&pkgInfo); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			fatalf("decode go list output: %v", err)
		}
		writeValue(h, "import", pkgInfo.ImportPath)
		writeModule(h, pkgInfo.Module)
		if !withinRoot(pkgInfo.Dir, root) {
			continue
		}
		for _, names := range [][]string{
			pkgInfo.GoFiles, pkgInfo.CgoFiles, pkgInfo.CFiles, pkgInfo.CXXFiles,
			pkgInfo.MFiles, pkgInfo.HFiles, pkgInfo.FFiles, pkgInfo.SFiles,
			pkgInfo.SwigFiles, pkgInfo.SwigCXXFiles, pkgInfo.SysoFiles, pkgInfo.EmbedFiles,
		} {
			for _, name := range names {
				files[filepath.Join(pkgInfo.Dir, name)] = struct{}{}
			}
		}
		if pkgInfo.Module != nil {
			addModuleFiles(files, pkgInfo.Module, root)
		}
	}
	if err := command.Wait(); err != nil {
		fatalf("go list failed: %v", err)
	}
}

func collectGoEnvironment(h hash.Hash, dir string) {
	command := exec.Command(
		"go", "env", "-json",
		"GOOS", "GOARCH", "GOAMD64", "GOARM", "GOARM64", "GO386", "GOMIPS", "GOMIPS64", "GOPPC64", "GORISCV64",
		"CGO_ENABLED", "CC", "CXX", "AR", "PKG_CONFIG", "GOEXPERIMENT", "GOFLAGS", "GOWORK",
	)
	command.Dir = dir
	output, err := command.Output()
	if err != nil {
		fatalf("read Go environment: %v", err)
	}
	var values map[string]string
	if err := json.Unmarshal(output, &values); err != nil {
		fatalf("decode Go environment: %v", err)
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		writeValue(h, "goenv:"+key, values[key])
	}
}

func writeModule(h hash.Hash, value *module) {
	if value == nil {
		return
	}
	writeValue(h, "module", value.Path+"\x00"+value.Version+"\x00"+value.Sum)
	if value.Replace != nil {
		writeValue(h, "replace", value.Replace.Path+"\x00"+value.Replace.Version+"\x00"+value.Replace.Sum)
	}
}

func addModuleFiles(files map[string]struct{}, value *module, root string) {
	for current := value; current != nil; current = current.Replace {
		if current.GoMod == "" || !withinRoot(current.GoMod, root) {
			continue
		}
		files[current.GoMod] = struct{}{}
		addIfPresent(files, filepath.Join(filepath.Dir(current.GoMod), "go.sum"))
	}
}

func collectExtra(files map[string]struct{}, path string) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		fatalf("resolve extra input %s: %v", path, err)
	}
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			files[absPath] = struct{}{}
			return
		}
		fatalf("inspect extra input %s: %v", path, err)
	}
	if !info.IsDir() {
		files[absPath] = struct{}{}
		return
	}
	err = filepath.WalkDir(absPath, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() && current != absPath {
			if entry.Name() == "node_modules" || entry.Name() == ".git" || entry.Name() == "dist" {
				return filepath.SkipDir
			}
			return nil
		}
		if !entry.IsDir() {
			files[current] = struct{}{}
		}
		return nil
	})
	if err != nil {
		fatalf("walk extra input %s: %v", path, err)
	}
}

func hashFiles(h hash.Hash, files map[string]struct{}, root string) {
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		label := path
		if relative, err := filepath.Rel(root, path); err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			label = filepath.ToSlash(relative)
		}
		writeValue(h, "file", label)
		file, err := os.Open(path)
		if err != nil {
			if os.IsNotExist(err) {
				writeValue(h, "missing", label)
				continue
			}
			fatalf("open input %s: %v", path, err)
		}
		if _, err := io.Copy(h, file); err != nil {
			_ = file.Close()
			fatalf("hash input %s: %v", path, err)
		}
		if err := file.Close(); err != nil {
			fatalf("close input %s: %v", path, err)
		}
	}
}

func addIfPresent(files map[string]struct{}, path string) {
	if _, err := os.Stat(path); err == nil {
		files[path] = struct{}{}
	}
}

func withinRoot(path, root string) bool {
	relative, err := filepath.Rel(root, path)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func writeValue(h hash.Hash, key, value string) {
	_, _ = io.WriteString(h, key)
	_, _ = h.Write([]byte{0})
	_, _ = io.WriteString(h, value)
	_, _ = h.Write([]byte{0})
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "dev-build-fingerprint: "+format+"\n", args...)
	os.Exit(1)
}
