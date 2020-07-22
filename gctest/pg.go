// Copyright © 2019 Martin Tournoij – This file is part of GoatCounter and
// published under the terms of a slightly modified EUPL v1.2 license, which can
// be found in the LICENSE file or at https://license.goatcounter.com

// +build testpg

package gctest

import (
	"zgo.at/goatcounter/cfg"
)

func init() {
	cfg.PgSQL = true
}
