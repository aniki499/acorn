package build

import (
	"fmt"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/flagparams"
	cplatforms "github.com/containerd/containerd/platforms"
)

func ParsePlatforms(platforms []string) (result []v1.Platform, _ error) {
	for _, platformString := range platforms {
		p, err := cplatforms.Parse(platformString)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", platformString, err)
		}
		result = append(result, v1.Platform(p))
	}
	return
}

func ParseParams(file, cwd string, args []string) (map[string]any, error) {
	appDefinition, err := ResolveAndParse(file, cwd)
	if err != nil {
		return nil, err
	}

	params, err := appDefinition.Args()
	if err != nil {
		return nil, err
	}

	return flagparams.New(ResolveFile(file, cwd), params).Parse(args)
}
