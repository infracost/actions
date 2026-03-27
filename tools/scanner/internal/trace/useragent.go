package trace

import (
	"fmt"

	"github.com/infracost/actions/tools/scanner/internal/version"
)

var (
	UserAgent = fmt.Sprintf("infracost-ci-%s", version.Version)
)
