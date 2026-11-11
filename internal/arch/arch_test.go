package arch

import (
	_ "embed"
	"encoding/json"
	"io"
	"os/exec"
	"strings"
	"testing"
)

//go:embed rules.json
var rulesJSON []byte

type layerRule struct {
	Prefix string   `json:"prefix"`
	Allow  []string `json:"allow"`
}

type ruleSet struct {
	Module string      `json:"module"`
	Layers []layerRule `json:"layers"`
}

type packageInfo struct {
	ImportPath string
	Imports    []string
}

func TestPackageDAG(t *testing.T) {
	var rules ruleSet
	if err := json.Unmarshal(rulesJSON, &rules); err != nil {
		t.Fatal(err)
	}

	for _, pkg := range listPackages(t, rules.Module) {
		layer, ok := matchLayer(rules, pkg.ImportPath)
		if !ok {
			continue
		}
		for _, imp := range pkg.Imports {
			if !strings.HasPrefix(imp, rules.Module+"/") {
				continue
			}
			if allowed(rules, layer.Allow, imp) {
				continue
			}
			t.Errorf("%s imports %s, which layer %s does not allow", pkg.ImportPath, imp, layer.Prefix)
		}
	}
}

func listPackages(t *testing.T, module string) []packageInfo {
	t.Helper()
	cmd := exec.Command("go", "list", "-json", module+"/...")
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	var packages []packageInfo
	decoder := json.NewDecoder(out)
	for {
		var pkg packageInfo
		err := decoder.Decode(&pkg)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		packages = append(packages, pkg)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}
	return packages
}

func matchLayer(rules ruleSet, importPath string) (layerRule, bool) {
	var best layerRule
	found := false
	for _, layer := range rules.Layers {
		target := rules.Module + "/" + layer.Prefix
		if importPath == target || strings.HasPrefix(importPath, target+"/") {
			if !found || len(layer.Prefix) > len(best.Prefix) {
				best = layer
				found = true
			}
		}
	}
	return best, found
}

func allowed(rules ruleSet, allow []string, imp string) bool {
	for _, entry := range allow {
		prefix := rules.Module + "/" + entry
		if imp == prefix || strings.HasPrefix(imp, prefix+"/") {
			return true
		}
	}
	return false
}
