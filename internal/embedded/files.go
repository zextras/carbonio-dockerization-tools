// SPDX-FileCopyrightText: 2026 Zextras <https://www.zextras.com>
//
// SPDX-License-Identifier: AGPL-3.0-only

package embedded

import "embed"

//go:embed all:carbonio-dockerization
var EmbeddedFiles embed.FS
