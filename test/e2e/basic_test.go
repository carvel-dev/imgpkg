package e2e

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestPushPull(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}}

	assetsPath := "assets/simple-app"
	path := "/tmp/imgpkg-test-basic"

	cleanUp := func() { os.RemoveAll(path) }
	cleanUp()
	defer cleanUp()

	// Git does not keep empty dir, hence create now
	err := os.MkdirAll(filepath.Join(assetsPath, "empty-dir"), 0700)
	if err != nil {
		t.Fatalf("Mkdir empty-dir")
	}

	imgpkg.Run([]string{"push", "-b", env.Image, "-f", assetsPath})
	imgpkg.Run([]string{"pull", "-i", env.Image, "-o", path})

	expectedFiles := []string{
		"README.md",
		"LICENSE",
		"config/config.yml",
		"config/inner-dir/README.txt",
	}

	for _, file := range expectedFiles {
		compareFiles(filepath.Join(assetsPath, file), filepath.Join(path, file), t)
	}

	err = ioutil.WriteFile(filepath.Join(path, "empty-dir", "file.txt"), []byte{}, 0600)
	if err != nil {
		t.Fatalf("Trying to write into empty dir")
	}
}

func TestPushMultipleFiles(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}}

	assetsPath := "assets/simple-app"
	path := "/tmp/imgpkg-test-push-multiple-files"

	cleanUp := func() { os.RemoveAll(path) }
	cleanUp()
	defer cleanUp()

	imgpkg.Run([]string{
		"push", "-b", env.Image,
		"-f", filepath.Join(assetsPath, "LICENSE"),
		"-f", filepath.Join(assetsPath, "README.md"),
		"-f", filepath.Join(assetsPath, "config"),
	})

	imgpkg.Run([]string{"pull", "-i", env.Image, "-o", path})

	expectedFiles := map[string]string{
		"README.md":                   "README.md",
		"LICENSE":                     "LICENSE",
		"config/config.yml":           "config.yml",
		"config/inner-dir/README.txt": "inner-dir/README.txt",
	}

	for assetFile, downloadedFile := range expectedFiles {
		compareFiles(filepath.Join(assetsPath, assetFile), filepath.Join(path, downloadedFile), t)
	}
}

func compareFiles(path1, path2 string, t *testing.T) {
	path1Bs, err := ioutil.ReadFile(path1)
	if err != nil {
		t.Fatalf("reading path1: %s", err)
	}

	path2Bs, err := ioutil.ReadFile(path2)
	if err != nil {
		t.Fatalf("reading path2: %s", err)
	}

	if string(path1Bs) != string(path2Bs) {
		t.Fatalf("Expected contents to match for %s vs %s", path1, path2)
	}
}
