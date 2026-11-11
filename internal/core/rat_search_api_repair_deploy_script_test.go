package core

import (
    "os"
    "os/exec"
    "path/filepath"
    "runtime"
    "strings"
    "testing"
)

func TestRATSearchAPIRepairDeployIsNarrowAndRollbackProtected(t *testing.T) {
    _, here, _, ok := runtime.Caller(0)
    if !ok {
        t.Fatal("cannot locate test source")
    }
    root := filepath.Clean(filepath.Join(filepath.Dir(here), "..", ".."))
    path := filepath.Join(root, "scripts", "ops", "deploy-rat-search-api-repair.sh")
    body, err := os.ReadFile(path)
    if err != nil {
        t.Fatal(err)
    }
    text := string(body)

    for _, want := range []string{
        "DEV_NS=\"rum-dev-isolated\"",
        "RAT_NS=\"rum-rate-anything-preview\"",
        "LIVE_NS=\"rum-dev\"",
        "DEV_HOST=\"dev-rum.daisycloversoftware.uk\"",
        "RAT_HOST=\"dev-rum-ra.daisycloversoftware.uk\"",
        "set image deployment/rum-api",
        "rollout status deployment/rum-api",
        "rat_frontend_preserved=",
        "rat_preview_api_preserved=",
        "live_unchanged=true",
        "rat-catalogue/search?q=$query",
        "Access-Control-Allow-Origin",
        "verify_search DJI",
        "verify_search CJ \"CJ Investigates\"",
        "restoring prior isolated DEV API",
    } {
        if !strings.Contains(text, want) {
            t.Fatalf("RAT search repair deployer missing %q", want)
        }
    }

    for _, forbidden := range []string{
        "set image deployment/rum-web",
        "set image deployment/rum-worker",
        "set image deployment/rum-rate-anything",
        "-n \"$live_ns\" set",
        "helm upgrade",
        "helm install",
        "kubectl apply",
        "kubectl delete",
        "insecure",
        "-k ",
        "--insecure",
    } {
        if strings.Contains(strings.ToLower(text), strings.ToLower(forbidden)) {
            t.Fatalf("RAT search repair deployer contains forbidden pattern %q", forbidden)
        }
    }

    if runtime.GOOS != "windows" {
        if _, err := exec.LookPath("bash"); err == nil {
            cmd := exec.Command("bash", "-n", path)
            if out, err := cmd.CombinedOutput(); err != nil {
                t.Fatalf("bash -n failed: %v: %s", err, out)
            }
        }
    }
}
