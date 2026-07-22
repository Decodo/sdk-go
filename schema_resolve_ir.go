package decodo

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type latestIRLocation struct {
	Version string
	URL     string
}

type gcsObjectList struct {
	Items []struct {
		Name string `json:"name"`
	} `json:"items"`
}

var irObjectPattern = regexp.MustCompile(`^decodo-ir-v(.+)\.json$`)

func parseSemver(version string) (major, minor, patch int, ok bool) {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return
	}
	var err error
	if major, err = strconv.Atoi(parts[0]); err != nil {
		return
	}
	if minor, err = strconv.Atoi(parts[1]); err != nil {
		return
	}
	if patch, err = strconv.Atoi(parts[2]); err != nil {
		return
	}
	ok = true
	return
}

func compareSemver(left, right string) int {
	lMaj, lMin, lPatch, lOk := parseSemver(left)
	rMaj, rMin, rPatch, rOk := parseSemver(right)
	if !lOk || !rOk {
		return strings.Compare(left, right)
	}
	if lMaj != rMaj {
		if lMaj > rMaj {
			return 1
		}
		return -1
	}
	if lMin != rMin {
		if lMin > rMin {
			return 1
		}
		return -1
	}
	if lPatch != rPatch {
		if lPatch > rPatch {
			return 1
		}
		return -1
	}
	return 0
}

func resolveLatestIR() (*latestIRLocation, error) {
	resp, err := schemaHTTPClient.Get(defaultIRListURL)
	if err != nil {
		return nil, fmt.Errorf("listing IR versions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("listing IR versions: HTTP %d", resp.StatusCode)
	}

	var list gcsObjectList
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, fmt.Errorf("decoding IR list: %w", err)
	}

	var versions []string
	for _, item := range list.Items {
		m := irObjectPattern.FindStringSubmatch(item.Name)
		if m == nil {
			continue
		}
		ver := m[1]
		if _, _, _, ok := parseSemver(ver); ok {
			versions = append(versions, ver)
		}
	}

	if len(versions) == 0 {
		return nil, fmt.Errorf("no versioned IR objects found in bucket with prefix %q", defaultIRPrefix)
	}

	latest := versions[0]
	for _, v := range versions[1:] {
		if compareSemver(v, latest) > 0 {
			latest = v
		}
	}

	return &latestIRLocation{
		Version: latest,
		URL:     fmt.Sprintf("%s/%s%s.json", defaultIRBase, defaultIRPrefix, latest),
	}, nil
}
